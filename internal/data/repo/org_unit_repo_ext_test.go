package repo

import "testing"

func TestOrgUnitTenantMatches_TreatsNilAndZeroAsPlatformDomain(t *testing.T) {
	zero := uint32(0)
	tenant := uint32(101)

	if !orgUnitTenantMatches(nil, nil) {
		t.Fatalf("expected nil/nil to match")
	}
	if !orgUnitTenantMatches(nil, &zero) {
		t.Fatalf("expected nil/0 to match as platform domain")
	}
	if !orgUnitTenantMatches(&zero, nil) {
		t.Fatalf("expected 0/nil to match as platform domain")
	}
	if !orgUnitTenantMatches(&zero, &zero) {
		t.Fatalf("expected 0/0 to match as platform domain")
	}
	if orgUnitTenantMatches(nil, &tenant) {
		t.Fatalf("expected nil/tenant to mismatch")
	}
	if orgUnitTenantMatches(&zero, &tenant) {
		t.Fatalf("expected 0/tenant to mismatch")
	}
	if !orgUnitTenantMatches(&tenant, &tenant) {
		t.Fatalf("expected same tenant ids to match")
	}
}
