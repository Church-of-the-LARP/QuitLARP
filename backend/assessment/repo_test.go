package assessment

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// mustWrite writes content to path, creating parent directories.
func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// writeTree writes every entry of files below root; keys are slash paths.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for path, content := range files {
		mustWrite(t, filepath.Join(root, filepath.FromSlash(path)), content)
	}
}

// writeValidTree writes a complete, valid assessment tree below root. It
// includes hidden and _build files that must be ignored by Load.
func writeValidTree(t *testing.T, root string) {
	t.Helper()
	mustWrite(t, filepath.Join(root, "Dockerfile"), "FROM alpine\n")
	mustWrite(t, filepath.Join(root, "README.md"), "# Assessment\n\nSolve the chapters.\n")
	mustWrite(t, filepath.Join(root, "dune-project"), "(lang dune 3.0)\n")
	mustWrite(t, filepath.Join(root, "chapters", ".gitkeep"), "")
	mustWrite(t, filepath.Join(root, "chapters", "1_a", "README.md"), "# Chapter a\n")
	mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"), `{"start": "clean", "task": "helper.ml"}`)
	mustWrite(t, filepath.Join(root, "chapters", "1_a", "a.ml"), "let () = ()\n[@@utest]\n")
	mustWrite(t, filepath.Join(root, "chapters", "1_a", "helper.ml"), "let x = 1\n")
	mustWrite(t, filepath.Join(root, "chapters", "1_a", "sub", "b.ml"), "module B = struct end\n[@@ utest]\n")
	mustWrite(t, filepath.Join(root, "chapters", "1_a", ".hidden.ml"), "let () = ()\n[@@utest]\n")
	mustWrite(t, filepath.Join(root, "chapters", "1_a", "_build", "gen.ml"), "let () = ()\n[@@utest]\n")
	mustWrite(t, filepath.Join(root, "chapters", "2_b", "README.md"), "# Chapter b\n")
	mustWrite(t, filepath.Join(root, "chapters", "2_b", "chapter.json"), `{"start": "continue", "task": "spec.ml", "note": "extra field"}`)
	mustWrite(t, filepath.Join(root, "chapters", "2_b", "dune-project"), "(lang dune 3.1)\n")
	mustWrite(t, filepath.Join(root, "chapters", "2_b", "spec.ml"), "let () = ()\n[@@utest]\n")
}

func TestLoadValid(t *testing.T) {
	root := t.TempDir()
	writeValidTree(t, root)

	content, violations := Load(root)
	if len(violations) != 0 {
		t.Fatalf("violations = %v, want none", violations)
	}
	if content.Description != "# Assessment\n\nSolve the chapters.\n" {
		t.Errorf("Description = %q", content.Description)
	}
	if content.Kind != "leetcode" {
		t.Errorf("Kind = %q, want leetcode", content.Kind)
	}
	if len(content.Chapters) != 2 {
		t.Fatalf("chapters = %d, want 2", len(content.Chapters))
	}

	first := content.Chapters[0]
	if first.Index != 1 || first.Name != "a" || first.Dir != "chapters/1_a" {
		t.Errorf("chapter 0 = %+v", first)
	}
	if first.Readme != "# Chapter a\n" {
		t.Errorf("chapter 0 Readme = %q", first.Readme)
	}
	if first.Start != "clean" {
		t.Errorf("chapter 0 Start = %q, want clean", first.Start)
	}
	if first.Task != "helper.ml" {
		t.Errorf("chapter 0 Task = %q, want helper.ml", first.Task)
	}
	if want := []string{"a.ml", "sub/b.ml"}; !reflect.DeepEqual(first.Specs, want) {
		t.Errorf("chapter 0 Specs = %v, want %v", first.Specs, want)
	}

	second := content.Chapters[1]
	if second.Index != 2 || second.Name != "b" || second.Dir != "chapters/2_b" {
		t.Errorf("chapter 1 = %+v", second)
	}
	if second.Start != "continue" {
		t.Errorf("chapter 1 Start = %q, want continue", second.Start)
	}
	if second.Task != "spec.ml" {
		t.Errorf("chapter 1 Task = %q, want spec.ml", second.Task)
	}
	if want := []string{"spec.ml"}; !reflect.DeepEqual(second.Specs, want) {
		t.Errorf("chapter 1 Specs = %v, want %v", second.Specs, want)
	}
}

func TestLoadKind(t *testing.T) {
	cases := []struct {
		name string
		json string
		file bool
		kind string
		want string
	}{
		{
			name: "assessment.json absent defaults to leetcode",
			kind: "leetcode",
		},
		{
			name: "explicit leetcode",
			file: true,
			json: `{"kind": "leetcode"}`,
			kind: "leetcode",
		},
		{
			name: "object without kind defaults to leetcode",
			file: true,
			json: `{}`,
			kind: "leetcode",
		},
		{
			name: "unknown kind",
			file: true,
			json: `{"kind": "x"}`,
			kind: "leetcode",
			want: `assessment.json: unknown kind "x"`,
		},
		{
			name: "kind is not a string",
			file: true,
			json: `{"kind": 7}`,
			kind: "leetcode",
			want: "assessment.json: unknown kind 7",
		},
		{
			name: "malformed assessment.json",
			file: true,
			json: `{`,
			kind: "leetcode",
			want: "assessment.json: invalid JSON: unexpected end of JSON input",
		},
		{
			name: "assessment.json is not an object",
			file: true,
			json: `[]`,
			kind: "leetcode",
			want: "assessment.json: not a JSON object",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeValidTree(t, root)
			if tc.file {
				mustWrite(t, filepath.Join(root, "assessment.json"), tc.json)
			}

			content, violations := Load(root)
			if tc.want == "" {
				if len(violations) != 0 {
					t.Fatalf("violations = %v, want none", violations)
				}
			} else if want := []string{tc.want}; !reflect.DeepEqual(violations, want) {
				t.Fatalf("violations = %v, want %v", violations, want)
			}
			if content.Kind != tc.kind {
				t.Errorf("Kind = %q, want %q", content.Kind, tc.kind)
			}
		})
	}
}

func TestLoadTaskPath(t *testing.T) {
	root := t.TempDir()
	writeValidTree(t, root)
	mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"),
		`{"start": "clean", "task": "./sub/b.ml"}`)

	content, violations := Load(root)
	if len(violations) != 0 {
		t.Fatalf("violations = %v, want none", violations)
	}
	if got := content.Chapters[0].Task; got != "sub/b.ml" {
		t.Errorf("chapter 0 Task = %q, want sub/b.ml", got)
	}
}

func TestLoadViolations(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(t *testing.T, root string)
		want   string
	}{
		{
			name: "missing Dockerfile",
			mutate: func(t *testing.T, root string) {
				mustRemove(t, filepath.Join(root, "Dockerfile"))
			},
			want: "Dockerfile: missing",
		},
		{
			name: "symlinked Dockerfile",
			mutate: func(t *testing.T, root string) {
				target := filepath.Join(root, "Dockerfile")
				mustRemove(t, target)
				if err := os.Symlink(filepath.Join(root, "README.md"), target); err != nil {
					t.Fatalf("symlink: %v", err)
				}
			},
			want: "Dockerfile: not a regular file",
		},
		{
			name: "blank root README",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "README.md"), "  \n\t\n")
			},
			want: "README.md: empty",
		},
		{
			name: "missing chapters",
			mutate: func(t *testing.T, root string) {
				mustRemoveAll(t, filepath.Join(root, "chapters"))
			},
			want: "chapters: missing",
		},
		{
			name: "stray visible file in chapters",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "chapters", "stray.txt"), "hi\n")
			},
			want: "chapters/stray.txt: not a directory",
		},
		{
			name: "name without underscore",
			mutate: func(t *testing.T, root string) {
				mustMkdir(t, filepath.Join(root, "chapters", "intro"))
			},
			want: "chapters/intro: chapter directory name must match <index>_<name>",
		},
		{
			name: "leading zero index",
			mutate: func(t *testing.T, root string) {
				mustMkdir(t, filepath.Join(root, "chapters", "01_c"))
			},
			want: "chapters/01_c: chapter directory name must match <index>_<name>",
		},
		{
			name: "invalid name characters",
			mutate: func(t *testing.T, root string) {
				mustMkdir(t, filepath.Join(root, "chapters", "3_bad!"))
			},
			want: "chapters/3_bad!: chapter directory name must match <index>_<name>",
		},
		{
			name: "duplicate index",
			mutate: func(t *testing.T, root string) {
				mustMkdir(t, filepath.Join(root, "chapters", "1_dup"))
			},
			want: "chapters/1_dup: duplicate chapter index 1 (already used by chapters/1_a)",
		},
		{
			name: "gap in indices",
			mutate: func(t *testing.T, root string) {
				mustRename(t, filepath.Join(root, "chapters", "2_b"), filepath.Join(root, "chapters", "3_c"))
			},
			want: "chapters: missing chapter index 2",
		},
		{
			name: "missing chapter README",
			mutate: func(t *testing.T, root string) {
				mustRemove(t, filepath.Join(root, "chapters", "1_a", "README.md"))
			},
			want: "chapters/1_a/README.md: missing",
		},
		{
			name: "missing chapter.json",
			mutate: func(t *testing.T, root string) {
				mustRemove(t, filepath.Join(root, "chapters", "1_a", "chapter.json"))
			},
			want: "chapters/1_a/chapter.json: missing",
		},
		{
			name: "invalid chapter.json",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"), "{")
			},
			want: "chapters/1_a/chapter.json: invalid JSON: unexpected end of JSON input",
		},
		{
			name: "chapter.json is not an object",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"), "[]")
			},
			want: "chapters/1_a/chapter.json: not a JSON object",
		},
		{
			name: "unknown start value",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"), `{"start": "maybe"}`)
			},
			want: `chapters/1_a/chapter.json: start must be "clean" or "continue", got maybe`,
		},
		{
			name: "first chapter is continue",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"),
					`{"start": "continue", "task": "helper.ml"}`)
			},
			want: `chapters/1_a/chapter.json: the first chapter must use start "clean"`,
		},
		{
			name: "task field missing",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"), `{"start": "clean"}`)
			},
			want: "chapters/1_a/chapter.json: task field is missing",
		},
		{
			name: "task is not a string",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"), `{"start": "clean", "task": 5}`)
			},
			want: "chapters/1_a/chapter.json: task must be a string, got 5",
		},
		{
			name: "task is empty",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"), `{"start": "clean", "task": ""}`)
			},
			want: "chapters/1_a/chapter.json: task must not be empty",
		},
		{
			name: "task is absolute",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"),
					`{"start": "clean", "task": "/etc/passwd"}`)
			},
			want: "chapters/1_a/chapter.json: task must be a relative path",
		},
		{
			name: "task escapes the chapter",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"),
					`{"start": "clean", "task": "../helper.ml"}`)
			},
			want: "chapters/1_a/chapter.json: task must stay inside the chapter directory",
		},
		{
			name: "task names a directory",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"),
					`{"start": "clean", "task": "sub"}`)
			},
			want: `chapters/1_a/chapter.json: task "sub": not a regular file`,
		},
		{
			name: "task names a missing file",
			mutate: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"),
					`{"start": "clean", "task": "glue/two_sum_functions.py"}`)
			},
			want: `chapters/1_a/chapter.json: task "glue/two_sum_functions.py": missing`,
		},
		{
			name: "chapter without a utest spec",
			mutate: func(t *testing.T, root string) {
				mustRemove(t, filepath.Join(root, "chapters", "1_a", "a.ml"))
				mustRemove(t, filepath.Join(root, "chapters", "1_a", "sub", "b.ml"))
			},
			want: "chapters/1_a: no .ml file declares a [@@utest] module",
		},
		{
			name: "chapter without any dune-project",
			mutate: func(t *testing.T, root string) {
				mustRemove(t, filepath.Join(root, "dune-project"))
			},
			want: "chapters/1_a: no dune-project at the chapter or an ancestor",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeValidTree(t, root)
			tc.mutate(t, root)

			_, violations := Load(root)
			if want := []string{tc.want}; !reflect.DeepEqual(violations, want) {
				t.Fatalf("violations = %v, want %v", violations, want)
			}
		})
	}
}

func mustRemove(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove %s: %v", path, err)
	}
}

func mustRemoveAll(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("removeall %s: %v", path, err)
	}
}

func mustRename(t *testing.T, from, to string) {
	t.Helper()
	if err := os.Rename(from, to); err != nil {
		t.Fatalf("rename %s: %v", from, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
