package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/vdplabs/opswatch/internal/domain"
)

func WriteJSON(w io.Writer, alerts []domain.Alert) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(alerts)
}

func WriteText(w io.Writer, alerts []domain.Alert) error {
	if len(alerts) == 0 {
		_, err := fmt.Fprintln(w, "No OpsWatch alerts.")
		return err
	}

	for _, alert := range alerts {
		if _, err := fmt.Fprintf(w, "[%s] %s: %s\n", strings.ToUpper(string(alert.Severity)), alert.Timestamp.Format("15:04:05"), alert.Title); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  %s\n", alert.Explanation); err != nil {
			return err
		}
		for _, evidence := range alert.Evidence {
			if _, err := fmt.Fprintf(w, "  - %s\n", evidence); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "  confidence: %.2f\n\n", alert.Confidence); err != nil {
			return err
		}
	}
	return nil
}
