package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDocumentsStandaloneFile(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "main.pwn")
	if err := os.WriteFile(source, []byte("/// Starts the mode.\npublic OnGameModeInit() { return 1; }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--project", source, "--format", "json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), root) {
		t.Fatalf("output contains absolute path: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"file": "main.pwn"`) {
		t.Fatalf("output = %s", stdout.String())
	}
}

func TestRunDocumentsDirectoryWithoutManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "api.inc"), []byte("/// Does work.\nnative Work();\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--project", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "## Work") {
		t.Fatalf("output = %s", stdout.String())
	}
}

func TestRunStrictFailsOnDiagnostics(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "api.inc")
	if err := os.WriteFile(source, []byte("native Undocumented();\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--project", source, "--strict"}, &stdout, &stderr); code != 1 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr.String(), "has no documentation") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestRunRejectsUnknownFormatBeforeReadingProject(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--project", "missing", "--format", "pdf"}, &stdout, &stderr); code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr.String(), `unknown output format "pdf"`) {
		t.Fatalf("stderr = %s", stderr.String())
	}
}
