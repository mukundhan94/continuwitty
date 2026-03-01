package models

import "testing"

func TestParseEngramLinkRelationType(t *testing.T) {
	valid := []EngramLinkRelationType{
		EngramLinkRelationSupports,
		EngramLinkRelationDependsOn,
		EngramLinkRelationContradicts,
		EngramLinkRelationRelatedTo,
		EngramLinkRelationDerivedFrom,
	}
	for _, relation := range valid {
		relation := relation
		t.Run(string(relation), func(t *testing.T) {
			parsed, err := ParseEngramLinkRelationType(string(relation))
			if err != nil {
				t.Fatalf("parse relation: %v", err)
			}
			if parsed != relation {
				t.Fatalf("expected relation %q, got %q", relation, parsed)
			}
		})
	}
	if _, err := ParseEngramLinkRelationType("invalid"); err == nil {
		t.Fatalf("expected invalid relation to fail")
	}
}

func TestParseEngramLinkOrigin(t *testing.T) {
	valid := []EngramLinkOrigin{
		EngramLinkOriginManual,
		EngramLinkOriginSuggested,
		EngramLinkOriginInferred,
		EngramLinkOriginSystem,
	}
	for _, origin := range valid {
		origin := origin
		t.Run(string(origin), func(t *testing.T) {
			parsed, err := ParseEngramLinkOrigin(string(origin))
			if err != nil {
				t.Fatalf("parse origin: %v", err)
			}
			if parsed != origin {
				t.Fatalf("expected origin %q, got %q", origin, parsed)
			}
		})
	}
	if _, err := ParseEngramLinkOrigin("invalid"); err == nil {
		t.Fatalf("expected invalid origin to fail")
	}
}

func TestParseEngramLinkStatus(t *testing.T) {
	valid := []EngramLinkStatus{
		EngramLinkStatusActive,
		EngramLinkStatusSuggested,
		EngramLinkStatusArchived,
		EngramLinkStatusRejected,
	}
	for _, status := range valid {
		status := status
		t.Run(string(status), func(t *testing.T) {
			parsed, err := ParseEngramLinkStatus(string(status))
			if err != nil {
				t.Fatalf("parse status: %v", err)
			}
			if parsed != status {
				t.Fatalf("expected status %q, got %q", status, parsed)
			}
		})
	}
	if _, err := ParseEngramLinkStatus("invalid"); err == nil {
		t.Fatalf("expected invalid status to fail")
	}
}
