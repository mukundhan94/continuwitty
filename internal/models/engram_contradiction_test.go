package models

import "testing"

func TestParseContradictionAlertStatus(t *testing.T) {
	valid := []ContradictionAlertStatus{
		ContradictionAlertStatusOpen,
		ContradictionAlertStatusResolved,
		ContradictionAlertStatusDismissed,
	}
	for _, status := range valid {
		status := status
		t.Run(string(status), func(t *testing.T) {
			parsed, err := ParseContradictionAlertStatus(string(status))
			if err != nil {
				t.Fatalf("parse status: %v", err)
			}
			if parsed != status {
				t.Fatalf("expected status %q, got %q", status, parsed)
			}
		})
	}
	if _, err := ParseContradictionAlertStatus("invalid"); err == nil {
		t.Fatalf("expected invalid status to fail")
	}
}
