package sessions

import "testing"

func TestParseChecks(t *testing.T) {
	output := `ok   adds up to the target
ok   the answer is not the first pair
FAIL equal values do not reuse an index: expected [0; 1], got [1; 1]
1 check(s) failed
`
	results := parseChecks(output)
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	if !results[0].Passed || results[0].Name != "adds up to the target" {
		t.Fatalf("first result = %+v", results[0])
	}
	if results[2].Passed {
		t.Fatalf("third result should have failed: %+v", results[2])
	}
	if results[2].Detail != "expected [0; 1], got [1; 1]" {
		t.Fatalf("failure detail = %q", results[2].Detail)
	}
}

func TestParseChecksIgnoresNoise(t *testing.T) {
	output := "dune: building...\nFatal error: exception Utf_bridge.Error(\"boom\")\n"
	if results := parseChecks(output); len(results) != 0 {
		t.Fatalf("got %d results from noise, want 0", len(results))
	}
}
