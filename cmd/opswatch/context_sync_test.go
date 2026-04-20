package main

import "testing"

func TestParseAWSCallerIdentity(t *testing.T) {
	identity, err := parseAWSCallerIdentity([]byte(`{"Account":"123456789012","Arn":"arn:aws:iam::123456789012:user/demo","UserId":"AIDA"}`))
	if err != nil {
		t.Fatal(err)
	}
	if identity.Account != "123456789012" {
		t.Fatalf("unexpected account %q", identity.Account)
	}
}

func TestParseAWSRoute53HostedZones(t *testing.T) {
	zones, err := parseAWSRoute53HostedZones([]byte(`{"HostedZones":[{"Id":"/hostedzone/Z123","Name":"example.com."},{"Id":"Z456","Name":"internal.example.com"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(zones) != 2 {
		t.Fatalf("expected two zones, got %d", len(zones))
	}
	if zones[0].ID != "Z123" || zones[0].Name != "example.com" {
		t.Fatalf("unexpected zone: %#v", zones[0])
	}
	if zones[1].ID != "Z456" || zones[1].Name != "internal.example.com" {
		t.Fatalf("unexpected zone: %#v", zones[1])
	}
}
