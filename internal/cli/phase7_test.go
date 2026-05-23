package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionString_DevDefault(t *testing.T) {
	if Version != "dev" {
		t.Skipf("version is %q (not dev — binary was built with ldflags)", Version)
	}
	want := "dev (commit: none, built: unknown)"
	got := rootCmd.Version
	if got != want {
		t.Errorf("version string = %q, want %q", got, want)
	}
}

func TestVersionString_ContainsVersion(t *testing.T) {
	v := rootCmd.Version
	if v == "" {
		t.Error("rootCmd.Version must not be empty")
	}
	if !strings.Contains(v, Version) {
		t.Errorf("version string %q does not contain Version %q", v, Version)
	}
	if !strings.Contains(v, "commit:") {
		t.Errorf("version string %q missing 'commit:' field", v)
	}
	if !strings.Contains(v, "built:") {
		t.Errorf("version string %q missing 'built:' field", v)
	}
}

func TestVersionFlag_PrintsVersion(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--version"})
	defer rootCmd.SetArgs(nil)
	defer rootCmd.SetOut(nil)

	rootCmd.Execute() //nolint:errcheck
	out := buf.String()
	if !strings.Contains(out, "lore") {
		t.Errorf("--version output %q does not contain 'lore'", out)
	}
	if !strings.Contains(out, Version) {
		t.Errorf("--version output %q does not contain Version %q", out, Version)
	}
}

func TestBinarySize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping binary size test in short mode")
	}

	dir := t.TempDir()
	out := filepath.Join(dir, "lore-test-binary")

	cmd := exec.Command("go", "build", "-ldflags=-s -w", "-o", out, "./cmd/lore")
	cmd.Dir = filepath.Join(repoRoot(t))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat binary: %v", err)
	}

	const maxBytes = 15 * 1024 * 1024 // 15 MB
	if info.Size() > maxBytes {
		t.Errorf("binary size %d bytes exceeds limit %d bytes (%.1f MB)",
			info.Size(), maxBytes, float64(info.Size())/(1024*1024))
	} else {
		t.Logf("binary size: %.1f MB (limit: 15 MB)", float64(info.Size())/(1024*1024))
	}
}

// repoRoot walks up from the test file to find go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod")
		}
		dir = parent
	}
}
