package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vdplabs/opswatch/internal/contextpack"
)

func runContextSync(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("context sync requires a provider, currently: aws")
	}
	switch args[0] {
	case "aws":
		return runContextSyncAWS(ctx, args[1:])
	default:
		return fmt.Errorf("unsupported context sync provider %q", args[0])
	}
}

func runContextSyncAWS(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("context sync aws", flag.ContinueOnError)
	contextDir := fs.String("context-dir", defaultContextDir(), "directory for local context packs")
	profile := fs.String("profile", "", "AWS CLI profile")
	region := fs.String("region", "", "AWS CLI region")
	environment := fs.String("environment", "", "environment label for the account, such as prod or staging")
	accountName := fs.String("account-name", "", "friendly account name; defaults to AWS account id")
	owner := fs.String("owner", "", "owning team for imported resources")
	risk := fs.String("risk", "", "risk label for imported resources")
	includeRoute53 := fs.Bool("include-route53", false, "import Route 53 hosted zones as protected domains")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*contextDir) == "" {
		return fmt.Errorf("--context-dir is required")
	}

	identity, err := awsCallerIdentity(ctx, awsCLIOptions{Profile: *profile, Region: *region})
	if err != nil {
		return err
	}
	name := firstNonEmptyValue(*accountName, identity.Account)
	pack := contextpack.Pack{
		AWSAccounts: []contextpack.AWSAccount{{
			ID:          identity.Account,
			Name:        name,
			Environment: *environment,
			Owner:       *owner,
			Risk:        *risk,
		}},
	}

	if *includeRoute53 {
		zones, err := awsRoute53HostedZones(ctx, awsCLIOptions{Profile: *profile, Region: *region})
		if err != nil {
			return err
		}
		for _, zone := range zones {
			pack.ProtectedDomains = append(pack.ProtectedDomains, contextpack.ProtectedDomain{
				Name:                zone.Name,
				Environment:         *environment,
				Owner:               *owner,
				AuthoritativeZoneID: zone.ID,
				Risk:                *risk,
			})
		}
	}

	path := filepath.Join(*contextDir, fmt.Sprintf("aws-%s.yaml", identity.Account))
	if err := contextpack.SaveYAML(path, pack); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "synced AWS context: %s\n", path)
	return nil
}

type awsCLIOptions struct {
	Profile string
	Region  string
}

type awsIdentity struct {
	Account string `json:"Account"`
	Arn     string `json:"Arn"`
	UserID  string `json:"UserId"`
}

type awsHostedZone struct {
	ID   string
	Name string
}

func awsCallerIdentity(ctx context.Context, options awsCLIOptions) (awsIdentity, error) {
	output, err := runAWS(ctx, options, "sts", "get-caller-identity")
	if err != nil {
		return awsIdentity{}, err
	}
	return parseAWSCallerIdentity(output)
}

func parseAWSCallerIdentity(output []byte) (awsIdentity, error) {
	var identity awsIdentity
	if err := json.Unmarshal(output, &identity); err != nil {
		return awsIdentity{}, err
	}
	if strings.TrimSpace(identity.Account) == "" {
		return awsIdentity{}, fmt.Errorf("aws sts get-caller-identity did not return an account id")
	}
	return identity, nil
}

func awsRoute53HostedZones(ctx context.Context, options awsCLIOptions) ([]awsHostedZone, error) {
	output, err := runAWS(ctx, options, "route53", "list-hosted-zones")
	if err != nil {
		return nil, err
	}
	return parseAWSRoute53HostedZones(output)
}

func parseAWSRoute53HostedZones(output []byte) ([]awsHostedZone, error) {
	var response struct {
		HostedZones []struct {
			ID   string `json:"Id"`
			Name string `json:"Name"`
		} `json:"HostedZones"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}
	zones := make([]awsHostedZone, 0, len(response.HostedZones))
	for _, zone := range response.HostedZones {
		name := strings.TrimSuffix(strings.TrimSpace(zone.Name), ".")
		if name == "" {
			continue
		}
		zones = append(zones, awsHostedZone{
			ID:   strings.TrimPrefix(zone.ID, "/hostedzone/"),
			Name: name,
		})
	}
	return zones, nil
}

func runAWS(ctx context.Context, options awsCLIOptions, args ...string) ([]byte, error) {
	allArgs := make([]string, 0, len(args)+6)
	allArgs = append(allArgs, args...)
	allArgs = append(allArgs, "--output", "json")
	if options.Profile != "" {
		allArgs = append(allArgs, "--profile", options.Profile)
	}
	if options.Region != "" {
		allArgs = append(allArgs, "--region", options.Region)
	}

	cmd := exec.CommandContext(ctx, "aws", allArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("aws %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func firstNonEmptyValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
