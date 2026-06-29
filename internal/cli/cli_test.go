package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunPrintsVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(context.Background(), []string{"--version"}, &stdout, &stderr, "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "v1.2.3\n" {
		t.Fatalf("got stdout %q, want %q", got, "v1.2.3\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("got stderr %q, want empty", stderr.String())
	}
}

func TestRunDefaultsToDev(t *testing.T) {
	var stdout bytes.Buffer

	err := Run(context.Background(), []string{"--version"}, &stdout, &bytes.Buffer{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "dev\n" {
		t.Fatalf("got stdout %q, want %q", got, "dev\n")
	}
}

func TestRunStillRequiresVaultAndConfig(t *testing.T) {
	err := Run(context.Background(), nil, &bytes.Buffer{}, &bytes.Buffer{}, "v1.2.3")
	if err == nil {
		t.Fatal("expected missing vault error")
	}
	if !strings.Contains(err.Error(), "--vault is required") {
		t.Fatalf("got error %q", err)
	}
}
