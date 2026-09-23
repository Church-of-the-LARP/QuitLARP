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
	mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"), `{"start": "clean"}`)
	mustWrite(t, filepath.Join(root, "chapters", "1_a", "a.ml"), "let () = ()\n[@@utest]\n")
	mustWrite(t, filepath.Join(root, "chapters", "1_a", "helper.ml"), "let x = 1\n")
	mustWrite(t, filepath.Join(root, "chapters", "1_a", "sub", "b.ml"), "module B = struct end\n[@@ utest]\n")
	mustWrite(t, filepath.Join(root, "chapters", "1_a", ".hidden.ml"), "let () = ()\n[@@utest]\n")
	mustWrite(t, filepath.Join(root, "chapters", "1_a", "_build", "gen.ml"), "let () = ()\n[@@utest]\n")
	mustWrite(t, filepath.Join(root, "chapters", "2_b", "README.md"), "# Chapter b\n")
	mustWrite(t, filepath.Join(root, "chapters", "2_b", "chapter.json"), `{"start": "continue", "note": "extra field"}`)
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
	if want := []string{"spec.ml"}; !reflect.DeepEqual(second.Specs, want) {
		t.Errorf("chapter 1 Specs = %v, want %v", second.Specs, want)
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
				mustWrite(t, filepath.Join(root, "chapters", "1_a", "chapter.json"), `{"start": "continue"}`)
			},
			want: `chapters/1_a/chapter.json: the first chapter must use start "clean"`,
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
