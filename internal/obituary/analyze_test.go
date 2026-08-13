package obituary

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type repositoryFixture struct {
	path         string
	indexBytes   []byte
	indexContent []byte
	worktree     fileState
}

type fileState struct {
	bytes []byte
	mode  os.FileMode
}

type restoreFixture struct {
	path            string
	indexedContent  []byte
	worktreeContent []byte
	mode            os.FileMode
}

func TestAnalyzeRestoreDotReportsOverwrittenUnstagedContent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := restoreFixture{
		path:            "note.txt",
		indexedContent:  []byte("committed line\n"),
		worktreeContent: []byte("committed line\nunstaged line\n"),
		mode:            0o644,
	}

	analyzed := buildRepository(t, ctx, fixture)
	oracle := buildRepository(t, ctx, fixture)

	assertRelevantStateEqual(t, analyzed, oracle)

	before := snapshotFile(t, filepath.Join(oracle.path, fixture.path))
	beforeIndex := append([]byte(nil), oracle.indexBytes...)
	runGit(t, ctx, oracle.path, "restore", ".")
	after := snapshotFile(t, filepath.Join(oracle.path, fixture.path))

	if !bytes.Equal(before.bytes, fixture.worktreeContent) {
		t.Fatalf("oracle before bytes = %q, want modified worktree bytes %q", before.bytes, fixture.worktreeContent)
	}
	if before.mode != fixture.mode {
		t.Fatalf("oracle before mode = %v, want %v", before.mode, fixture.mode)
	}
	if bytes.Equal(before.bytes, after.bytes) && before.mode == after.mode {
		t.Fatal("real git restore did not overwrite the oracle worktree state")
	}
	if !bytes.Equal(after.bytes, fixture.indexedContent) {
		t.Fatalf("oracle after bytes = %q, want index bytes %q", after.bytes, fixture.indexedContent)
	}
	if after.mode != fixture.mode {
		t.Fatalf("oracle after mode = %v, want %v", after.mode, fixture.mode)
	}
	if len(beforeIndex) == 0 {
		t.Fatal("oracle index snapshot is empty")
	}

	t.Log("fixture and real-Git oracle validated; reaching the expected Analyze red sentinel")

	// RED: Task 2 replaces this sentinel with a call through the pre-agreed seam:
	//
	//	report := Analyze(ctx, analyzed.path, []string{"git", "restore", "."})
	//
	// and compares the report's sole casualty with the independently derived
	// before state above. Keeping the sentinel at runtime lets this test prove
	// that fixture construction and the real-Git oracle succeed first.
	t.Fatal("RED: Analyze(context.Context, cwd, argv) Report behavior is absent")
}

func buildRepository(t *testing.T, ctx context.Context, fixture restoreFixture) repositoryFixture {
	t.Helper()

	dir := t.TempDir()
	runGit(t, ctx, dir, "-c", "init.defaultBranch=main", "init", "--quiet")

	path := filepath.Join(dir, fixture.path)
	if err := os.WriteFile(path, fixture.indexedContent, fixture.mode); err != nil {
		t.Fatalf("write indexed fixture content: %v", err)
	}
	if err := os.Chmod(path, fixture.mode); err != nil {
		t.Fatalf("set indexed fixture mode: %v", err)
	}

	runGit(t, ctx, dir, "add", "--", fixture.path)
	runGit(t, ctx, dir,
		"-c", "user.name=Obituary Test",
		"-c", "user.email=obituary@example.invalid",
		"commit", "--quiet", "-m", "fixture baseline",
	)

	if err := os.WriteFile(path, fixture.worktreeContent, fixture.mode); err != nil {
		t.Fatalf("write modified fixture content: %v", err)
	}
	if err := os.Chmod(path, fixture.mode); err != nil {
		t.Fatalf("set modified fixture mode: %v", err)
	}

	indexBytes, err := os.ReadFile(filepath.Join(dir, ".git", "index"))
	if err != nil {
		t.Fatalf("read exact index file bytes: %v", err)
	}
	indexContent := runGitOutput(t, ctx, dir, "show", "--no-textconv", ":"+fixture.path)

	return repositoryFixture{
		path:         dir,
		indexBytes:   indexBytes,
		indexContent: indexContent,
		worktree:     snapshotFile(t, path),
	}
}

func assertRelevantStateEqual(t *testing.T, left, right repositoryFixture) {
	t.Helper()

	if !bytes.Equal(left.worktree.bytes, right.worktree.bytes) {
		t.Fatalf("paired worktree bytes differ: %q != %q", left.worktree.bytes, right.worktree.bytes)
	}
	if left.worktree.mode != right.worktree.mode {
		t.Fatalf("paired worktree modes differ: %v != %v", left.worktree.mode, right.worktree.mode)
	}
	if !bytes.Equal(left.indexContent, right.indexContent) {
		t.Fatalf("paired index content differs: %q != %q", left.indexContent, right.indexContent)
	}
	if len(left.indexBytes) == 0 || len(right.indexBytes) == 0 {
		t.Fatal("paired repository index snapshot is empty")
	}
}

func snapshotFile(t *testing.T, path string) fileState {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("%q mode = %v, want regular file", path, info.Mode())
	}

	return fileState{bytes: content, mode: info.Mode()}
}

func runGit(t *testing.T, ctx context.Context, dir string, argv ...string) {
	t.Helper()
	runGitOutput(t, ctx, dir, argv...)
}

func runGitOutput(t *testing.T, ctx context.Context, dir string, argv ...string) []byte {
	t.Helper()

	cmd := exec.CommandContext(ctx, "git", argv...)
	cmd.Dir = dir
	cmd.Env = []string{
		"LC_ALL=C",
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		"PATH=" + os.Getenv("PATH"),
	}
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("git %v in %q: %v\nstderr: %s", argv, dir, err, exitErr.Stderr)
		}
		t.Fatalf("git %v in %q: %v", argv, dir, err)
	}
	return output
}
