package bootstrap

import "testing"

func TestRegisteredHostModulesIncludesXDev(t *testing.T) {
	modules := registeredHostModules()
	if len(modules) == 0 {
		t.Fatalf("expected at least one registered host module")
	}

	found := false
	for _, module := range modules {
		if module == nil {
			continue
		}
		if module.Name() == "xdev" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected registered host modules to include xdev")
	}
}
