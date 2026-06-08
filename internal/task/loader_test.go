package task

import (
	"testing"
)

func TestNewRegistryFromRepos_LoadsDefaultExecutors(t *testing.T) {
	registry, err := NewRegistryFromRepos(nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewRegistryFromRepos failed: %v", err)
	}
	if _, ok := registry.Get("system:cleanup:audit-logs"); !ok {
		t.Fatalf("expected cleanup executor to be registered")
	}
	if _, ok := registry.Get("system:task:runtime-summary"); !ok {
		t.Fatalf("expected runtime summary executor to be registered")
	}
}
