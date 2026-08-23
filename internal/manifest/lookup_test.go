package manifest

import "testing"

func TestLookupServiceByComposeName(t *testing.T) {
	t.Parallel()

	m := validManifest()
	svc, err := m.LookupService("consultant_hub")
	if err != nil {
		t.Fatal(err)
	}
	if svc.ComposeService != "consultant_hub" || svc.SourcePort != 5021 {
		t.Fatalf("%#v", svc)
	}
}

func TestLookupServiceByUniqueRepoName(t *testing.T) {
	t.Parallel()

	m := validManifest()
	svc, err := m.LookupService("identity-api")
	if err != nil {
		t.Fatal(err)
	}
	if svc.ComposeService != "identity.api" {
		t.Fatalf("compose = %q", svc.ComposeService)
	}
}

func TestLookupServiceRejectsAmbiguousRepo(t *testing.T) {
	t.Parallel()

	m := validManifest()
	if _, err := m.LookupService("consultant-suite"); err == nil {
		t.Fatal("multi-service repo name must be ambiguous")
	}
}

func TestLookupServiceMissing(t *testing.T) {
	t.Parallel()

	m := validManifest()
	if _, err := m.LookupService("wallet"); err == nil {
		t.Fatal("expected missing service")
	}
	var nilM *Manifest
	if _, err := nilM.LookupService("doctor"); err == nil {
		t.Fatal("nil manifest")
	}
}
