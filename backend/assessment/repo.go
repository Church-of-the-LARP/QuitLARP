// Package assessment holds the domain half of push validation: it reads and
// checks the assessment tree, materializes a pushed commit into a temp
// directory, and runs binding generation over the chapter specs.
package assessment

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Content is an assessment tree. A non-empty violations slice from Load means
// the content is incomplete and the push must be rejected.
type Content struct {
	Description string
	Chapters    []Chapter
}

// Chapter is one chapter of the assessment. Dir is a slash path relative to the
// tree root, Name is the directory name without the index prefix, Start is the
// "start" value of chapter.json and Specs are chapter-relative slash paths of
// the .ml files that declare a [@@utest] module.
type Chapter struct {
	Index  int
	Name   string
	Dir    string
	Readme string
	Start  string
	Specs  []string
}

var (
	indexPattern = regexp.MustCompile(`^[1-9][0-9]*$`)
	namePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	utestPattern = regexp.MustCompile(`\[@@\s*utest\s*\]`)
)

// chapterRef is a chapter directory whose name parsed, before its contents are
// read.
type chapterRef struct {
	index int
	name  string
	entry string
}

// Load reads the tree at root, validates the assessment layout and returns what
// it parsed plus every violation. A non-empty violations slice means the push
// must be rejected.
func Load(root string) (Content, []string) {
	root = filepath.Clean(root)

	var content Content
	var violations []string
	add := func(format string, args ...any) {
		violations = append(violations, fmt.Sprintf(format, args...))
	}

	// Rules 1 and 2: the root Dockerfile and README.
	readFileAt(filepath.Join(root, "Dockerfile"), "Dockerfile", add)
	if data, ok := readFileAt(filepath.Join(root, "README.md"), "README.md", add); ok {
		if strings.TrimSpace(data) == "" {
			add("README.md: empty")
		} else {
			content.Description = data
		}
	}

	// Rule 3: the chapters directory and its direct visible entries.
	entries, ok := readChapterEntries(root, add)
	if !ok {
		return content, violations
	}

	// Rules 4 and 5: chapter directory names and the index sequence.
	var refs []chapterRef
	seen := map[int]string{}
	for _, name := range entries {
		index, chapterName, ok := parseChapterDir(name)
		if !ok {
			add("chapters/%s: chapter directory name must match <index>_<name>", name)
			continue
		}
		if prev, dup := seen[index]; dup {
			add("chapters/%s: duplicate chapter index %d (already used by chapters/%s)", name, index, prev)
			continue
		}
		seen[index] = name
		refs = append(refs, chapterRef{index: index, name: chapterName, entry: name})
	}
	for i := 1; i <= maxIndex(seen); i++ {
		if _, ok := seen[i]; !ok {
			add("chapters: missing chapter index %d", i)
		}
	}

	// Rules 6 to 10: the contents of every structurally valid chapter.
	chapters := make([]Chapter, 0, len(refs))
	for _, ref := range refs {
		ch := Chapter{
			Index: ref.index,
			Name:  ref.name,
			Dir:   "chapters/" + ref.entry,
		}
		loadChapter(root, ref.entry, &ch, add)
		chapters = append(chapters, ch)
	}
	sort.Slice(chapters, func(i, j int) bool { return chapters[i].Index < chapters[j].Index })
	content.Chapters = chapters
	return content, violations
}

// readChapterEntries checks the chapters directory and returns the names of its
// visible subdirectories, sorted. It returns false when the directory itself is
// unusable, in which case no chapter can be examined.
func readChapterEntries(root string, add func(string, ...any)) ([]string, bool) {
	dir := filepath.Join(root, "chapters")
	info, err := os.Lstat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			add("chapters: missing")
		} else {
			add("chapters: %v", err)
		}
		return nil, false
	}
	if !info.IsDir() {
		add("chapters: not a directory")
		return nil, false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		add("chapters: %v", err)
		return nil, false
	}

	var dirs []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !entry.IsDir() {
			add("chapters/%s: not a directory", name)
			continue
		}
		dirs = append(dirs, name)
	}
	if len(dirs) == 0 {
		add("chapters: no chapter directories")
		return nil, false
	}
	sort.Strings(dirs)
	return dirs, true
}

// loadChapter runs rules 6 to 10 for one chapter directory and fills in ch.
func loadChapter(root, entry string, ch *Chapter, add func(string, ...any)) {
	dir := filepath.Join(root, "chapters", entry)
	base := "chapters/" + entry

	// Rule 6: the chapter README.
	if data, ok := readFileAt(filepath.Join(dir, "README.md"), base+"/README.md", add); ok {
		if strings.TrimSpace(data) == "" {
			add("%s/README.md: empty", base)
		} else {
			ch.Readme = data
		}
	}

	// Rule 7: the chapter manifest.
	if data, ok := readFileAt(filepath.Join(dir, "chapter.json"), base+"/chapter.json", add); ok {
		start, err := parseStart(data)
		if err != nil {
			add("%s/chapter.json: %v", base, err)
		} else {
			ch.Start = start
		}
	}

	// Rule 8: the first chapter always assigns a clean file.
	if ch.Index == 1 && ch.Start == "continue" {
		add("%s/chapter.json: the first chapter must use start \"clean\"", base)
	}

	// Rule 9: at least one .ml file declaring a [@@utest] module.
	specs, err := findSpecs(dir)
	if err != nil {
		add("%s: %v", base, err)
	} else if len(specs) == 0 {
		add("%s: no .ml file declares a [@@utest] module", base)
	} else {
		ch.Specs = specs
	}

	// Rule 10: a dune-project at the chapter or any ancestor up to the root.
	if !hasDuneProject(root, dir) {
		add("%s: no dune-project at the chapter or an ancestor", base)
	}
}

// parseChapterDir splits a chapter directory name into its index and name.
func parseChapterDir(dir string) (int, string, bool) {
	i := strings.IndexByte(dir, '_')
	if i <= 0 || i == len(dir)-1 {
		return 0, "", false
	}
	indexText, name := dir[:i], dir[i+1:]
	if !indexPattern.MatchString(indexText) || !namePattern.MatchString(name) {
		return 0, "", false
	}
	index, err := strconv.Atoi(indexText)
	if err != nil {
		return 0, "", false
	}
	return index, name, true
}

// parseStart extracts the "start" field of a chapter.json manifest, which must
// be a JSON object whose start is exactly "clean" or "continue".
func parseStart(data string) (string, error) {
	var value any
	if err := json.Unmarshal([]byte(data), &value); err != nil {
		return "", fmt.Errorf("invalid JSON: %v", err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return "", errors.New("not a JSON object")
	}
	raw, ok := object["start"]
	if !ok {
		return "", errors.New("start field is missing")
	}
	start, ok := raw.(string)
	if !ok || (start != "clean" && start != "continue") {
		return "", fmt.Errorf("start must be \"clean\" or \"continue\", got %v", raw)
	}
	return start, nil
}

// findSpecs walks a chapter directory and returns the chapter-relative slash
// paths of the .ml files that declare a [@@utest] module, sorted.
func findSpecs(dir string) ([]string, error) {
	var specs []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if name == "_build" || strings.HasPrefix(name, ".") {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".ml") || !d.Type().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !utestPattern.Match(data) {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		specs = append(specs, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(specs)
	return specs, nil
}

// hasDuneProject reports whether a regular dune-project file exists in dir or in
// any ancestor of dir up to and including root.
func hasDuneProject(root, dir string) bool {
	root = filepath.Clean(root)
	for {
		info, err := os.Lstat(filepath.Join(dir, "dune-project"))
		if err == nil && info.Mode().IsRegular() {
			return true
		}
		if filepath.Clean(dir) == root {
			return false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// readFileAt reads a regular file and reports a violation when it is missing,
// is not a regular file (a symlink counts here) or cannot be read.
func readFileAt(path, loc string, add func(string, ...any)) (string, bool) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			add("%s: missing", loc)
		} else {
			add("%s: %v", loc, err)
		}
		return "", false
	}
	if !info.Mode().IsRegular() {
		add("%s: not a regular file", loc)
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		add("%s: %v", loc, err)
		return "", false
	}
	return string(data), true
}

// maxIndex returns the largest index seen, or 0 when there are none.
func maxIndex(seen map[int]string) int {
	max := 0
	for index := range seen {
		if index > max {
			max = index
		}
	}
	return max
}
