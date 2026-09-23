package assessment

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCommandGeneratorSuccess(t *testing.T) {
	g := CommandGenerator{Command: "echo"}
	report, err := g.Generate(context.Background(), t.TempDir(), []string{"spec-a.ml", "spec-b.ml"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, spec := range []string{"spec-a.ml", "spec-b.ml"} {
		if got := strings.Count(report, spec); got != 2 {
			t.Errorf("report mentions %q %d times, want 2 (header plus echo)\nreport:\n%s", spec, got, report)
		}
	}
}

func TestCommandGeneratorWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.ml"), "chapter-local\n")

	g := CommandGenerator{Command: "cat"}
	report, err := g.Generate(context.Background(), dir, []string{"a.ml"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(report, "chapter-local") {
		t.Errorf("report = %q, want the chapter-local file content", report)
	}
}

func TestCommandGeneratorFailure(t *testing.T) {
	g := CommandGenerator{Command: "false"}
	report, err := g.Generate(context.Background(), t.TempDir(), []string{"a.ml", "b.ml"})
	if err == nil {
		t.Fatal("Generate succeeded, want an error")
	}
	if want := "generation failed for 2 of 2 spec files"; err.Error() != want {
		t.Errorf("err = %q, want %q", err.Error(), want)
	}
	for _, spec := range []string{"a.ml", "b.ml"} {
		if !strings.Contains(report, spec+": exit status 1") {
			t.Errorf("report = %q, want a failure line for %s", report, spec)
		}
	}
}

func TestCommandGeneratorDisabled(t *testing.T) {
	g := CommandGenerator{}
	report, err := g.Generate(context.Background(), t.TempDir(), []string{"a.ml"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if report != "" {
		t.Errorf("report = %q, want empty", report)
	}
}

func TestCommandGeneratorTimeout(t *testing.T) {
	// The spec "5" keeps the command a plain sleep rather than an argument
	// error, so the deadline is what fails it.
	g := CommandGenerator{Command: "sleep 5", Timeout: 100 * time.Millisecond}
	start := time.Now()
	_, err := g.Generate(context.Background(), t.TempDir(), []string{"5"})
	if err == nil {
		t.Fatal("Generate succeeded, want a timeout error")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("Generate took %v, want it to respect the timeout", elapsed)
	}
}

func TestShellQuote(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"a.ml", "'a.ml'"},
		{"sub/b.ml", "'sub/b.ml'"},
		{"a b.ml", "'a b.ml'"},
		{"it's.ml", `'it'\''s.ml'`},
	}
	for _, tc := range cases {
		if got := shellQuote(tc.in); got != tc.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCapReport(t *testing.T) {
	small := "keep me\n"
	if got := capReport(small); got != small {
		t.Errorf("capReport(small) = %q, want unchanged", got)
	}

	long := strings.Repeat("x", maxReportBytes+100)
	got := capReport(long)
	if !strings.HasPrefix(got, "[output truncated]\n") {
		t.Errorf("capReport prefix = %q", got[:min(24, len(got))])
	}
	if !strings.HasSuffix(got, strings.Repeat("x", 100)) {
		t.Error("capReport did not keep the tail")
	}
	if len(got) != len("[output truncated]\n")+maxReportBytes {
		t.Errorf("capReport length = %d", len(got))
	}
}
