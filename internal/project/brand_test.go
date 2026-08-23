package project

import "testing"

func TestFromEnv(t *testing.T) {
	t.Setenv(EnvVar, " isos ")
	if got := FromEnv(); got != "isos" {
		t.Fatalf("FromEnv() = %q", got)
	}
}
