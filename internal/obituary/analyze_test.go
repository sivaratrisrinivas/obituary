package obituary

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type repositoryFixture struct {
	path         string
	indexBytes   []byte
	indexContent []byte
	refs         []byte
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

	report := Analyze(ctx, analyzed.path, []string{"git", "restore", "."})
	assertAnalysisReadOnly(t, ctx, analyzed, fixture.path)

	complete, ok := report.(CompleteReport)
	if !ok {
		t.Fatalf("Analyze outcome = %T, want CompleteReport", report)
	}
	casualties := complete.Casualties()
	if len(casualties) != 1 {
		t.Fatalf("casualty count = %d, want 1", len(casualties))
	}

	casualty := casualties[0]
	if casualty.Path() != fixture.path {
		t.Fatalf("casualty path = %q, want %q", casualty.Path(), fixture.path)
	}
	if !bytes.Equal(casualty.Content(), before.bytes) {
		t.Fatalf("casualty bytes = %q, want oracle before bytes %q", casualty.Content(), before.bytes)
	}
	wantExecutable := before.mode.Perm()&0o111 != 0
	if casualty.Executable() != wantExecutable {
		t.Fatalf("casualty executable = %t, want mode-compatible value from %v", casualty.Executable(), before.mode)
	}
	delta, ok := casualty.Delta().(TextDelta)
	if !ok {
		t.Fatalf("casualty delta = %T, want TextDelta", casualty.Delta())
	}
	if delta.Additions() != 1 || delta.Deletions() != 0 {
		t.Fatalf("casualty delta = +%d/-%d, want +1/-0", delta.Additions(), delta.Deletions())
	}
	negative, ok := casualty.Evidence().(NoExactSamePathCopyFound)
	if !ok {
		t.Fatalf("casualty evidence = %T, want NoExactSamePathCopyFound", casualty.Evidence())
	}
	if negative.Claim() != NoExactCopyClaim {
		t.Fatalf("negative claim = %q, want %q", negative.Claim(), NoExactCopyClaim)
	}
	if negative.NotChecked() != NotCheckedClaim {
		t.Fatalf("not-checked disclosure = %q, want %q", negative.NotChecked(), NotCheckedClaim)
	}
}

func TestAnalyzeRecognizesOnlyExactSupportedCommands(t *testing.T) {
	t.Parallel()

	supported := [][]string{
		{"git", "restore", "."},
		{"git", "restore", "--", "."},
		{"git", "checkout", "."},
		{"git", "checkout", "--", "."},
	}
	for _, argv := range supported {
		argv := append([]string(nil), argv...)
		t.Run(strings.Join(argv, " "), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			report := Analyze(ctx, t.TempDir(), argv)
			if _, ok := report.(SearchIncompleteReport); !ok {
				t.Fatalf("Analyze(%q) outcome = %T, want SearchIncompleteReport proving command recognition", argv, report)
			}
		})
	}

	unsupported := []struct {
		name string
		argv []string
	}{
		{name: "empty", argv: nil},
		{name: "different executable", argv: []string{"/usr/bin/git", "restore", "."}},
		{name: "different subcommand", argv: []string{"git", "reset", "."}},
		{name: "flag", argv: []string{"git", "restore", "--worktree", "."}},
		{name: "missing pathspec", argv: []string{"git", "restore"}},
		{name: "different pathspec", argv: []string{"git", "restore", "./note.txt"}},
		{name: "treeish", argv: []string{"git", "checkout", "HEAD", "--", "."}},
		{name: "extra argument", argv: []string{"git", "restore", ".", "other"}},
		{name: "wrong ordering", argv: []string{"git", "restore", ".", "--"}},
	}
	for _, test := range unsupported {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			report := Analyze(context.Background(), filepath.Join(t.TempDir(), "does-not-exist"), test.argv)
			unknown, ok := report.(UnknownReport)
			if !ok {
				t.Fatalf("Analyze(%q) outcome = %T, want UnknownReport", test.argv, report)
			}
			if unknown.Reason() != UnsupportedCommand {
				t.Fatalf("Analyze(%q) reason = %q, want %q", test.argv, unknown.Reason(), UnsupportedCommand)
			}
		})
	}
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
	refs := runGitOutput(t, ctx, dir, "for-each-ref", "--format=%(refname)%00%(objectname)%00")

	return repositoryFixture{
		path:         dir,
		indexBytes:   indexBytes,
		indexContent: indexContent,
		refs:         refs,
		worktree:     snapshotFile(t, path),
	}
}

func assertAnalysisReadOnly(t *testing.T, ctx context.Context, before repositoryFixture, relativePath string) {
	t.Helper()

	afterWorktree := snapshotFile(t, filepath.Join(before.path, relativePath))
	if !bytes.Equal(afterWorktree.bytes, before.worktree.bytes) || afterWorktree.mode != before.worktree.mode {
		t.Fatalf("Analyze mutated worktree state: before bytes/mode = %q/%v, after = %q/%v", before.worktree.bytes, before.worktree.mode, afterWorktree.bytes, afterWorktree.mode)
	}
	afterIndex, err := os.ReadFile(filepath.Join(before.path, ".git", "index"))
	if err != nil {
		t.Fatalf("read index after Analyze: %v", err)
	}
	if !bytes.Equal(afterIndex, before.indexBytes) {
		t.Fatal("Analyze mutated exact index bytes")
	}
	afterRefs := runGitOutput(t, ctx, before.path, "for-each-ref", "--format=%(refname)%00%(objectname)%00")
	if !bytes.Equal(afterRefs, before.refs) {
		t.Fatalf("Analyze mutated refs: before = %q, after = %q", before.refs, afterRefs)
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
