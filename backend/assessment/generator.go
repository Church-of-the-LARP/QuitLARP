package assessment

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// maxReportBytes caps the combined generator report. The tail is kept because
// the interesting failure output is usually at the end.
const maxReportBytes = 16 << 10

// Generator produces the UFT glue for a chapter project.
type Generator interface {
	// Generate runs the generation command for every spec file (paths relative
	// to chapterDir) and returns the combined tool output. A nil error means
	// every invocation succeeded.
	Generate(ctx context.Context, chapterDir string, specs []string) (string, error)
}

// CommandGenerator runs a shell command per spec file, with the spec path
// appended and the working directory set to the chapter.
type CommandGenerator struct {
	Command string
	Timeout time.Duration
}

// Generate runs Command once per spec, appending the shell-quoted spec path.
// Every spec is attempted even when an earlier one fails; the returned error is
// non-nil when at least one invocation failed.
func (g CommandGenerator) Generate(ctx context.Context, chapterDir string, specs []string) (string, error) {
	if g.Command == "" {
		return "", nil
	}

	var report strings.Builder
	failed := 0
	for _, spec := range specs {
		report.WriteString(spec + "\n")

		runCtx := ctx
		var cancel context.CancelFunc
		if g.Timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, g.Timeout)
		}
		line := g.Command + " " + shellQuote(spec)
		cmd := exec.CommandContext(runCtx, "sh", "-c", line)
		cmd.Dir = chapterDir
		if g.Timeout > 0 {
			// The command runs under a shell, so a killed shell can leave a
			// grandchild holding the output pipe; bound how long Wait blocks.
			cmd.WaitDelay = g.Timeout
		}
		out, err := cmd.CombinedOutput()
		if cancel != nil {
			cancel()
		}
		report.Write(out)
		if len(out) > 0 && out[len(out)-1] != '\n' {
			report.WriteByte('\n')
		}
		if err != nil {
			failed++
			report.WriteString(fmt.Sprintf("%s: %v\n", spec, err))
		}
	}

	text := capReport(report.String())
	if failed > 0 {
		return text, fmt.Errorf("generation failed for %d of %d spec files", failed, len(specs))
	}
	return text, nil
}

// capReport truncates a report from the front, keeping the tail.
func capReport(report string) string {
	if len(report) <= maxReportBytes {
		return report
	}
	keep := report[len(report)-maxReportBytes:]
	return "[output truncated]\n" + keep
}

// shellQuote single-quotes a path for sh, escaping embedded single quotes.
func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}
