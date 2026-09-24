package sessions

import (
	"regexp"
	"strings"
)

// The UFT harness prints one line per check: "ok   <name>" on success and
// "FAIL <name>: <detail>" on failure (see the check helper in the chapter
// specs).
var (
	okLine   = regexp.MustCompile(`^ok\s+(.+?)\s*$`)
	failLine = regexp.MustCompile(`^FAIL\s+([^:]+):\s*(.*)$`)
)

// parseChecks reads check results out of one harness run. Lines that are not
// results (build noise, tracebacks) are ignored; the raw output stays
// available to the caller.
func parseChecks(output string) []TestResult {
	var results []TestResult
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if match := okLine.FindStringSubmatch(line); match != nil {
			results = append(results, TestResult{Name: match[1], Passed: true})
			continue
		}
		if match := failLine.FindStringSubmatch(line); match != nil {
			results = append(results, TestResult{
				Name:   strings.TrimSpace(match[1]),
				Passed: false,
				Detail: strings.TrimSpace(match[2]),
			})
		}
	}
	return results
}
