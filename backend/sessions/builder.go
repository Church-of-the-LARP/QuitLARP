package sessions

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"backend/assessment"
	"backend/config"
	"backend/supervisor"
)

// Builder turns one assessment commit into a runnable environment: the
// author's image plus a /workspace holding the repository and the compiled
// UFT harness. Results are cached per assessment and commit, so the first
// session pays the build cost and later ones start immediately.
type Builder struct {
	sup *supervisor.Supervisor
	cfg *config.Config

	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewBuilder(sup *supervisor.Supervisor, cfg *config.Config) *Builder {
	return &Builder{sup: sup, cfg: cfg, locks: map[string]*sync.Mutex{}}
}

// Build returns the environment for a commit, building it when the cache or
// the image is missing. Calls for the same commit are serialized.
func (b *Builder) Build(ctx context.Context, assessmentID int64, repoDir, sha string) (*Environment, error) {
	unlock := b.lock(fmt.Sprintf("%d:%s", assessmentID, sha))
	defer unlock()

	cacheDir := filepath.Join(b.cfg.AssessmentEnv.DataDir, "assessments",
		fmt.Sprintf("assessment-%d", assessmentID), sha)
	metaPath := filepath.Join(cacheDir, "environment.json")
	treeDir := filepath.Join(cacheDir, "tree")

	if env, err := loadEnvironment(metaPath); err == nil {
		if ok, err := b.sup.HasImage(ctx, env.Image); err == nil && ok {
			return env, nil
		}
	}

	if err := ensureTree(ctx, treeDir, repoDir, sha); err != nil {
		return nil, err
	}
	content, violations := assessment.Load(treeDir)
	if len(violations) > 0 {
		return nil, fmt.Errorf("assessment %d at %s violates the layout: %s",
			assessmentID, shortSHA(sha), strings.Join(violations, "; "))
	}

	if err := b.ensureHarness(ctx); err != nil {
		return nil, err
	}

	baseTag := fmt.Sprintf("%s-base-%d:%s", b.cfg.AssessmentEnv.ImagePrefix, assessmentID, shortSHA(sha))
	if err := b.ensureImage(ctx, baseTag, []contextTree{{Root: treeDir}}, nil); err != nil {
		return nil, fmt.Errorf("build the assessment image: %w", err)
	}

	if _, err := os.Stat(filepath.Join(treeDir, "_build")); err != nil {
		if err := b.compile(ctx, treeDir); err != nil {
			return nil, err
		}
	}

	chapters, err := discoverChapters(treeDir, content)
	if err != nil {
		return nil, err
	}

	finalTag := fmt.Sprintf("%s-%d:%s", b.cfg.AssessmentEnv.ImagePrefix, assessmentID, shortSHA(sha))
	dockerfile := []byte("FROM " + baseTag + "\nCOPY tree /workspace\n")
	if err := b.ensureImage(ctx, finalTag, []contextTree{{Root: treeDir, Prefix: "tree", IncludeBuild: true}},
		map[string][]byte{"Dockerfile": dockerfile}); err != nil {
		return nil, fmt.Errorf("build the assessment environment image: %w", err)
	}

	env := &Environment{
		AssessmentID: assessmentID,
		SHA:          sha,
		Image:        finalTag,
		Dir:          treeDir,
		Chapters:     chapters,
	}
	if err := writeEnvironment(metaPath, env); err != nil {
		return nil, err
	}
	log.Printf("sessions: built environment for assessment %d at %s (%d chapters)",
		assessmentID, shortSHA(sha), len(chapters))
	return env, nil
}

// lock serializes builds of the same assessment commit.
func (b *Builder) lock(key string) func() {
	b.mu.Lock()
	lock, ok := b.locks[key]
	if !ok {
		lock = &sync.Mutex{}
		b.locks[key] = lock
	}
	b.mu.Unlock()
	lock.Lock()
	return lock.Unlock
}

// ensureTree materializes the commit into the cache when it is not there yet.
func ensureTree(ctx context.Context, treeDir, repoDir, sha string) error {
	if _, err := os.Stat(filepath.Join(treeDir, "Dockerfile")); err == nil {
		return nil
	}
	if err := os.RemoveAll(treeDir); err != nil {
		return err
	}
	return assessment.Materialize(ctx, repoDir, sha, "", treeDir)
}

// ensureHarness builds the platform's OCaml image once. The UFT libraries a
// chapter links against are vendored into each assessment build context, so
// this image only carries the compiler, dune and ppxlib.
func (b *Builder) ensureHarness(ctx context.Context) error {
	ok, err := b.sup.HasImage(ctx, b.cfg.AssessmentEnv.HarnessImage)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	log.Printf("sessions: building the harness image %s (first run, this can take a while)", b.cfg.AssessmentEnv.HarnessImage)
	contextTar, err := supervisor.HarnessImageContext()
	if err != nil {
		return fmt.Errorf("prepare the harness build context: %w", err)
	}
	buildCtx, cancel := context.WithTimeout(ctx, b.cfg.AssessmentEnv.BuildTimeout)
	defer cancel()
	if err := b.sup.BuildImage(buildCtx, b.cfg.AssessmentEnv.HarnessImage, contextTar, supervisor.HarnessDockerfilePath); err != nil {
		return fmt.Errorf("build the harness image: %w", err)
	}
	return nil
}

// ensureImage builds an image when it is missing. The build context holds the
// given trees plus extra files at its root, which is where the Dockerfile
// lives.
func (b *Builder) ensureImage(ctx context.Context, tag string, trees []contextTree, extra map[string][]byte) error {
	ok, err := b.sup.HasImage(ctx, tag)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	log.Printf("sessions: building image %s", tag)

	file, err := os.CreateTemp("", "assessment-context-*.tar")
	if err != nil {
		return err
	}
	contextPath := file.Name()
	defer os.Remove(contextPath)
	if err := writeContext(file, trees, extra); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	reader, err := os.Open(contextPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	buildCtx, cancel := context.WithTimeout(ctx, b.cfg.AssessmentEnv.BuildTimeout)
	defer cancel()
	return b.sup.BuildImage(buildCtx, tag, reader, "Dockerfile")
}

// compile builds the chapter harness of the materialized tree inside a
// container of the harness image and copies the result back, so the tree
// gains its _build directory.
//
// The UFT libraries (utf_core, utf_bridge, utf_ppx) are private dune
// libraries, so opam cannot install them; they are vendored into the build
// workspace under utf-vendor/ instead, where the chapter's dune stanzas
// resolve them like any other project library.
func (b *Builder) compile(ctx context.Context, treeDir string) error {
	log.Printf("sessions: compiling the assessment harness")

	utfSrc := b.cfg.AssessmentEnv.UTFSourceDir
	trees := []contextTree{
		{Root: treeDir, Prefix: "tree"},
		{Root: filepath.Join(utfSrc, "core"), Prefix: "tree/utf-vendor/core", SkipTestDirs: true},
		{Root: filepath.Join(utfSrc, "bridge"), Prefix: "tree/utf-vendor/bridge", SkipTestDirs: true},
		{Root: filepath.Join(utfSrc, "ppx"), Prefix: "tree/utf-vendor/ppx", SkipTestDirs: true},
	}
	for _, tree := range trees[1:] {
		if info, err := os.Stat(tree.Root); err != nil || !info.IsDir() {
			return fmt.Errorf("the vendored UFT sources are missing at %s", tree.Root)
		}
	}

	buildTag := fmt.Sprintf("%s-harnessbuild:%s", b.cfg.AssessmentEnv.ImagePrefix, shortSHA(hashOfDir(treeDir)))
	dockerfile := []byte("FROM " + b.cfg.AssessmentEnv.HarnessImage +
		"\nUSER root\nCOPY tree /work\nRUN chown -R opam:opam /work\nUSER opam" +
		"\nRUN cd /work && opam exec -- dune build @all\n")

	buildCtx, cancel := context.WithTimeout(ctx, b.cfg.AssessmentEnv.BuildTimeout)
	defer cancel()
	if err := b.ensureImage(buildCtx, buildTag, trees, map[string][]byte{"Dockerfile": dockerfile}); err != nil {
		return fmt.Errorf("compile the assessment harness: %w", err)
	}

	id := "harness-" + randomHex(8)
	spec := supervisor.ContainerSpec{
		Runtime:         "runsc",
		Cmd:             []string{"sleep", "infinity"},
		MemoryBytes:     2 << 30,
		NanoCPUs:        2_000_000_000,
		PidsLimit:       512,
		NetworkDisabled: true,
		Labels:          map[string]string{"codingtest.role": "assessment-harness"},
	}
	if err := b.sup.CreateContainer(buildTag, id, spec); err != nil {
		return err
	}
	defer b.sup.RemoveContainer(id)
	if err := b.sup.StartContainer(id); err != nil {
		return err
	}
	if err := b.sup.CopyFromDir(id, "/work", treeDir); err != nil {
		return fmt.Errorf("collect the compiled harness: %w", err)
	}

	// The vendored sources served their purpose during the build; keep them
	// out of the environment image.
	if err := os.RemoveAll(filepath.Join(treeDir, "utf-vendor")); err != nil {
		return err
	}
	return nil
}

// discoverChapters maps every chapter's spec files to the executables dune
// built for them.
func discoverChapters(treeDir string, content assessment.Content) ([]ChapterBuild, error) {
	builds := make([]ChapterBuild, 0, len(content.Chapters))
	for _, ch := range content.Chapters {
		build := ChapterBuild{Dir: ch.Dir, Title: ch.Name, Task: ch.Task}
		for _, spec := range ch.Specs {
			exe, err := findExe(treeDir, ch.Dir, spec)
			if err != nil {
				return nil, fmt.Errorf("chapter %s: %w", ch.Dir, err)
			}
			build.Specs = append(build.Specs, SpecRun{
				File:    path.Join(ch.Dir, spec),
				Exe:     exe,
				WorkDir: path.Dir(path.Join(ch.Dir, spec)),
			})
		}
		if len(build.Specs) == 0 {
			return nil, fmt.Errorf("chapter %s: no checks were found", ch.Dir)
		}
		builds = append(builds, build)
	}
	return builds, nil
}

// testNamePattern finds the test name of a dune stanza, e.g.
// "(test (name two_sum) ...)". The stanza body up to the name may not hold
// parentheses, which keeps one stanza from matching another's name.
var testNamePattern = regexp.MustCompile(`\(test\b[^()]*\(name\s+([A-Za-z0-9_.-]+)`)

// findExe locates the executable dune built for one spec file. It prefers
// the test name declared next to the spec, then the spec file's own name,
// and falls back to the single executable under the build directory.
func findExe(treeDir, chapterDir, spec string) (string, error) {
	specDir := path.Dir(path.Join(chapterDir, spec))
	buildDir := path.Join("_build/default", specDir)
	specBase := strings.TrimSuffix(path.Base(spec), ".ml")

	names := duneTestNames(filepath.Join(treeDir, filepath.FromSlash(specDir)))
	names = append(names, specBase)
	for _, name := range names {
		exe := path.Join(buildDir, name+".exe")
		if isRegularFile(filepath.Join(treeDir, filepath.FromSlash(exe))) {
			return exe, nil
		}
	}

	found, err := findExecutables(filepath.Join(treeDir, filepath.FromSlash(buildDir)))
	if err != nil {
		return "", err
	}
	for _, rel := range found {
		if strings.TrimSuffix(path.Base(rel), ".exe") == specBase {
			return path.Join(buildDir, rel), nil
		}
	}
	if len(found) == 1 {
		return path.Join(buildDir, found[0]), nil
	}
	if len(found) == 0 {
		return "", fmt.Errorf("no executable was built for %s", spec)
	}
	return "", fmt.Errorf("cannot tell which executable belongs to %s (found %s)",
		spec, strings.Join(found, ", "))
}

// duneTestNames reads the test names declared in the dune file of a
// directory.
func duneTestNames(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, "dune"))
	if err != nil {
		return nil
	}
	var names []string
	for _, match := range testNamePattern.FindAllStringSubmatch(string(data), -1) {
		names = append(names, match[1])
	}
	return names
}

// findExecutables lists the .exe files under root, relative to root.
func findExecutables(root string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || path.Ext(d.Name()) != ".exe" {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		found = append(found, filepath.ToSlash(rel))
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return found, nil
}

func isRegularFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}

// contextTree is one directory tree placed in a Docker build context.
type contextTree struct {
	Root         string
	Prefix       string // slash path of the tree inside the context; empty means the context root
	SkipTestDirs bool   // leave directories named "test" out (vendored sources carry their own suites)
	IncludeBuild bool   // keep _build directories; the environment image carries the compiled harness
}

// writeContext writes a Docker build context tar: every tree under its
// prefix, plus extra files at the context root. Build artifacts, VCS metadata
// and macOS metadata files never reach the context.
func writeContext(w io.Writer, trees []contextTree, extra map[string][]byte) error {
	tw := tar.NewWriter(w)
	for name, content := range extra {
		header := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tw.Write(content); err != nil {
			return err
		}
	}
	for _, tree := range trees {
		if err := writeContextTree(tw, tree); err != nil {
			return err
		}
	}
	return tw.Close()
}

func writeContextTree(tw *tar.Writer, tree contextTree) error {
	return filepath.WalkDir(tree.Root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			skip := name == ".git" || (name == "_build" && !tree.IncludeBuild) || (tree.SkipTestDirs && name == "test")
			if skip {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, "._") || name == ".DS_Store" {
			return nil
		}
		rel, err := filepath.Rel(tree.Root, p)
		if err != nil {
			return err
		}
		tarName := filepath.ToSlash(rel)
		if tree.Prefix != "" {
			tarName = tree.Prefix + "/" + tarName
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case info.Mode().IsRegular():
			header := &tar.Header{Name: tarName, Mode: int64(info.Mode().Perm()), Size: info.Size(), Typeflag: tar.TypeReg}
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			file, err := os.Open(p)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, file); err != nil {
				file.Close()
				return err
			}
			return file.Close()
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return tw.WriteHeader(&tar.Header{Name: tarName, Linkname: target, Mode: 0o777, Typeflag: tar.TypeSymlink})
		default:
			return nil
		}
	})
}

func loadEnvironment(path string) (*Environment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var env Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	if env.Image == "" || env.Dir == "" {
		return nil, fmt.Errorf("environment cache at %s is incomplete", path)
	}
	return &env, nil
}

func writeEnvironment(path string, env *Environment) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// hashOfDir fingerprints a tree by path, size and modification time, enough
// to tell one compile input from another.
func hashOfDir(root string) string {
	h := fnv.New64a()
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		fmt.Fprintf(h, "%s:%d:%d\n", p, info.Size(), info.ModTime().UnixNano())
		return nil
	})
	return fmt.Sprintf("%x", h.Sum64())
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
