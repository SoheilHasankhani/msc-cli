package dockerapi

import "testing"

func TestParseComposeServicesJSON(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"services": {
			"wallet": {"container_name": "isos-wallet", "image": "registry.isos.clinic/isos/wallet:latest"},
			"patient": {"container_name": "isos-patient", "image": "registry.isos.clinic/isos/patient:latest"},
			"db": {"image": "mcr.microsoft.com/mssql/server:2025-RC1-ubuntu-24.04"}
		}
	}`)
	services, err := parseComposeServicesJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 3 {
		t.Fatalf("len = %d", len(services))
	}
	if services[0].Name != "db" || services[0].ContainerName != "db" {
		t.Fatalf("db = %#v", services[0])
	}
	if services[1].Name != "patient" || services[1].ContainerName != "isos-patient" {
		t.Fatalf("patient = %#v", services[1])
	}
	if services[2].Name != "wallet" || services[2].Image == "" {
		t.Fatalf("wallet = %#v", services[2])
	}
}
