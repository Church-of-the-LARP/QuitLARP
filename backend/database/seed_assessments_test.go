package database

import "testing"

// TestSeededAssessmentTrees guards the embedded starter trees: every seeded
// assessment must pass the same layout validation a push does, or the app
// refuses to boot.
func TestSeededAssessmentTrees(t *testing.T) {
	seen := map[string]bool{}
	for _, seed := range seededAssessments {
		seed := seed
		t.Run(seed.Key, func(t *testing.T) {
			if seen[seed.Key] {
				t.Fatalf("duplicate seed key %q", seed.Key)
			}
			seen[seed.Key] = true

			files, err := seedFiles(seed.Key)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := files[seed.TemplateFile]; !ok {
				t.Fatalf("template %q is missing", seed.TemplateFile)
			}

			content, err := seedContent(seed.Key, files)
			if err != nil {
				t.Fatal(err)
			}
			if content.Description == "" {
				t.Fatal("README.md must provide a description")
			}

			switch seed.Key {
			case "twosum":
				if content.Kind != "leetcode" {
					t.Fatalf("kind = %q, want leetcode", content.Kind)
				}
				if len(content.Chapters) != 1 {
					t.Fatalf("chapters = %d, want 1", len(content.Chapters))
				}
			case "petrik-verseny":
				if content.Kind != "unbound" {
					t.Fatalf("kind = %q, want unbound", content.Kind)
				}
				if files["deploy/index.html"] == nil || files["deploy/style.css"] == nil {
					t.Fatal("deploy/index.html and deploy/style.css are required")
				}
			}
		})
	}
}
