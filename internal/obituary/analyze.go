package obituary

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

var errUnsupportedEvidence = errors.New("reachable exact bytes require mode and locator resolution")

// Analyze performs a read-only preflight for one of the four exact supported Git argv forms.
func Analyze(ctx context.Context, cwd string, argv []string) Report {
	if !supportedCommand(argv) {
		return newUnknownReport(argv, UnsupportedCommand)
	}
	if ctx.Err() != nil {
		return newSearchIncompleteReport(argv, SearchInterrupted)
	}

	root, err := repositoryRoot(ctx, cwd)
	if err != nil {
		return newUnknownReport(argv, InspectionFailed)
	}
	resolvedCWD, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return newUnknownReport(argv, InspectionFailed)
	}
	resolvedCWD, err = filepath.Abs(resolvedCWD)
	if err != nil || resolvedCWD != root || !slices.Equal(argv, []string{"git", "restore", "."}) {
		return newUnknownReport(argv, UnsupportedState)
	}
	if err := supportedRepositoryConfiguration(ctx, cwd); err != nil {
		return newUnknownReport(argv, UnsupportedState)
	}
	candidates, err := discoverCandidates(ctx, cwd)
	if err != nil {
		return newUnknownReport(argv, InspectionFailed)
	}
	if len(candidates) != 1 || candidates[0].indexMode != "100644" || candidates[0].workMode != "100644" {
		return newUnknownReport(argv, UnsupportedState)
	}
	if err := transformationsAreSafe(ctx, cwd, candidates); err != nil {
		return newUnknownReport(argv, InspectionFailed)
	}

	casualties, err := loadCasualties(ctx, root, cwd, candidates)
	if err != nil {
		return newUnknownReport(argv, InspectionFailed)
	}
	if _, ok := casualties[0].delta.(TextDelta); !ok {
		return newUnknownReport(argv, UnsupportedState)
	}
	if ctx.Err() != nil {
		return newSearchIncompleteReport(argv, SearchInterrupted)
	}
	if err := searchEvidence(ctx, cwd, casualties); err != nil {
		if ctx.Err() != nil {
			return newSearchIncompleteReport(argv, SearchInterrupted)
		}
		if errors.Is(err, errUnsupportedEvidence) {
			return newUnknownReport(argv, UnsupportedState)
		}
		return newSearchIncompleteReport(argv, InspectionFailed)
	}

	report, err := newCompleteReport(argv, root, casualties)
	if err != nil {
		return newUnknownReport(argv, InspectionFailed)
	}
	return report
}

func supportedCommand(argv []string) bool {
	if len(argv) == 3 {
		return argv[0] == "git" &&
			(argv[1] == "restore" || argv[1] == "checkout") &&
			argv[2] == "."
	}
	if len(argv) == 4 {
		return argv[0] == "git" &&
			(argv[1] == "restore" || argv[1] == "checkout") &&
			argv[2] == "--" && argv[3] == "."
	}
	return false
}

func repositoryRoot(ctx context.Context, cwd string) (string, error) {
	root, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	gitDirectory, err := os.Stat(filepath.Join(root, ".git"))
	if err != nil || !gitDirectory.IsDir() {
		return "", errors.New("Task 2 supports only a repository-root worktree with a .git directory")
	}
	result, err := inspectGit(ctx, root, nil, "rev-parse", "--is-inside-work-tree", "--is-bare-repository")
	if err != nil {
		return "", err
	}
	if string(result.stdout) != "true\nfalse\n" {
		return "", errors.New("not a normal non-bare worktree")
	}
	return root, nil
}

func supportedRepositoryConfiguration(ctx context.Context, cwd string) error {
	checks := []struct {
		key  string
		want string
	}{
		{key: "core.sparseCheckout", want: "false"},
		{key: "core.autocrlf", want: "false"},
		{key: "core.fileMode", want: "true"},
	}
	for _, check := range checks {
		result, err := inspectGit(ctx, cwd, nil, "config", "--type=bool", "--get", "--default="+check.want, check.key)
		if err != nil || strings.TrimSuffix(string(result.stdout), "\n") != check.want {
			return fmt.Errorf("unsupported %s configuration", check.key)
		}
	}
	return nil
}

type candidate struct {
	path      string
	indexMode string
	workMode  string
}

func discoverCandidates(ctx context.Context, cwd string) ([]candidate, error) {
	unmerged, err := inspectGit(ctx, cwd, nil, "ls-files", "--unmerged", "-z", "--", ".")
	if err != nil || len(unmerged.stdout) != 0 {
		return nil, errors.New("unmerged index is unsupported")
	}

	result, err := inspectGit(ctx, cwd, nil, "diff-files", "--no-ext-diff", "--no-textconv", "--raw", "-z", "--", ".")
	if err != nil {
		return nil, err
	}
	if len(result.stdout) == 0 {
		return nil, nil
	}

	fields := bytes.Split(result.stdout, []byte{0})
	var candidates []candidate
	for index := 0; index < len(fields)-1; {
		header := strings.Fields(string(fields[index]))
		if len(header) != 5 || header[4] != "M" || index+1 >= len(fields)-1 {
			return nil, errors.New("unsupported affected entry")
		}
		path := string(fields[index+1])
		if path == "" || strings.ContainsRune(path, '\n') || !regularGitMode(header[0]) || !regularGitMode(header[1]) {
			return nil, errors.New("unsupported affected path or mode")
		}
		candidates = append(candidates, candidate{
			path:      path,
			indexMode: strings.TrimPrefix(header[0], ":"),
			workMode:  header[1],
		})
		index += 2
	}
	return candidates, nil
}

func regularGitMode(mode string) bool {
	mode = strings.TrimPrefix(mode, ":")
	return mode == "100644" || mode == "100755"
}

func transformationsAreSafe(ctx context.Context, cwd string, candidates []candidate) error {
	var input bytes.Buffer
	for _, candidate := range candidates {
		input.WriteString(candidate.path)
		input.WriteByte(0)
	}
	result, err := inspectGit(ctx, cwd, input.Bytes(), "check-attr", "-z", "--stdin", "filter", "working-tree-encoding", "text", "eol")
	if err != nil {
		return err
	}
	parts := bytes.Split(result.stdout, []byte{0})
	if len(parts) == 0 || len(parts[len(parts)-1]) != 0 {
		return errors.New("invalid check-attr output")
	}
	parts = parts[:len(parts)-1]
	if len(parts) != len(candidates)*4*3 {
		return errors.New("incomplete check-attr output")
	}
	for index := 0; index < len(parts); index += 3 {
		attribute := string(parts[index+1])
		value := string(parts[index+2])
		switch attribute {
		case "filter", "working-tree-encoding", "eol":
			if value != "unspecified" {
				return fmt.Errorf("active %s transformation", attribute)
			}
		case "text":
			if value != "unspecified" && value != "unset" {
				return errors.New("active text transformation")
			}
		default:
			return errors.New("unexpected attribute")
		}
	}
	return nil
}

func loadCasualties(ctx context.Context, root, cwd string, candidates []candidate) ([]Casualty, error) {
	argv := []string{"ls-files", "--stage", "-z", "--"}
	for _, candidate := range candidates {
		argv = append(argv, candidate.path)
	}
	result, err := inspectGit(ctx, cwd, nil, argv...)
	if err != nil {
		return nil, err
	}
	entries := bytes.Split(result.stdout, []byte{0})
	if len(entries) != len(candidates)+1 || len(entries[len(entries)-1]) != 0 {
		return nil, errors.New("unexpected index entries")
	}

	casualties := make([]Casualty, 0, len(candidates))
	for index, candidate := range candidates {
		metadataAndPath := bytes.SplitN(entries[index], []byte{'\t'}, 2)
		if len(metadataAndPath) != 2 || string(metadataAndPath[1]) != candidate.path {
			return nil, errors.New("index path mismatch")
		}
		metadata := strings.Fields(string(metadataAndPath[0]))
		if len(metadata) != 3 || metadata[0] != candidate.indexMode || metadata[2] != "0" {
			return nil, errors.New("unsupported index entry")
		}

		path := filepath.Join(root, filepath.FromSlash(candidate.path))
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			return nil, errors.New("affected worktree entry is not a regular file")
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		delta, err := inspectDelta(ctx, cwd, candidate.path)
		if err != nil {
			return nil, err
		}
		casualties = append(casualties, Casualty{
			path:       candidate.path,
			content:    content,
			executable: candidate.workMode == "100755",
			delta:      delta,
		})
	}
	return casualties, nil
}

func inspectDelta(ctx context.Context, cwd, path string) (Delta, error) {
	result, err := inspectGit(ctx, cwd, nil, "diff-files", "--no-ext-diff", "--no-textconv", "--numstat", "-z", "--", path)
	if err != nil {
		return nil, err
	}
	parts := bytes.Split(result.stdout, []byte{0})
	if len(parts) != 2 || len(parts[1]) != 0 {
		return nil, errors.New("unexpected numstat output")
	}
	fields := bytes.Split(parts[0], []byte{'\t'})
	if len(fields) != 3 || string(fields[2]) != path {
		return nil, errors.New("numstat path mismatch")
	}
	if string(fields[0]) == "-" && string(fields[1]) == "-" {
		return binaryDelta{}, nil
	}
	additions, addErr := strconv.Atoi(string(fields[0]))
	deletions, deleteErr := strconv.Atoi(string(fields[1]))
	if addErr != nil || deleteErr != nil || additions < 0 || deletions < 0 {
		return nil, errors.New("invalid numstat counts")
	}
	return textDelta{additions: additions, deletions: deletions}, nil
}

func searchEvidence(ctx context.Context, cwd string, casualties []Casualty) error {
	indexObjects, indexModes, err := indexEvidenceObjects(ctx, cwd, casualties)
	if err != nil {
		return err
	}
	objects, err := inspectObjects(ctx, cwd, indexObjects)
	if err != nil {
		return err
	}
	for index := range casualties {
		object := objects[indexObjects[index]]
		if object.typ != "blob" {
			return errors.New("index object is not a blob")
		}
		if bytes.Equal(object.data, casualties[index].content) && (indexModes[index] == "100755") == casualties[index].executable {
			casualties[index].evidence = exactSamePathCopyFound{locator: "index:" + casualties[index].path}
		}
	}

	refs, err := localEvidenceRefs(ctx, cwd)
	if err != nil {
		return err
	}
	for index := range casualties {
		if casualties[index].evidence != nil {
			continue
		}
		found, err := exactBytesReachableAtPath(ctx, cwd, refs, casualties[index])
		if err != nil {
			return err
		}
		if found {
			return errUnsupportedEvidence
		}
		casualties[index].evidence = noExactSamePathCopyFound{}
	}
	return nil
}

func indexEvidenceObjects(ctx context.Context, cwd string, casualties []Casualty) ([]string, []string, error) {
	argv := []string{"ls-files", "--stage", "-z", "--"}
	for _, casualty := range casualties {
		argv = append(argv, casualty.path)
	}
	result, err := inspectGit(ctx, cwd, nil, argv...)
	if err != nil {
		return nil, nil, err
	}
	entries := bytes.Split(result.stdout, []byte{0})
	if len(entries) != len(casualties)+1 {
		return nil, nil, errors.New("unexpected evidence index entries")
	}
	objectIDs := make([]string, 0, len(casualties))
	modes := make([]string, 0, len(casualties))
	for index, casualty := range casualties {
		metadataAndPath := bytes.SplitN(entries[index], []byte{'\t'}, 2)
		if len(metadataAndPath) != 2 || string(metadataAndPath[1]) != casualty.path {
			return nil, nil, errors.New("evidence index path mismatch")
		}
		metadata := strings.Fields(string(metadataAndPath[0]))
		if len(metadata) != 3 || metadata[2] != "0" || !regularGitMode(metadata[0]) {
			return nil, nil, errors.New("unsupported evidence index entry")
		}
		objectIDs = append(objectIDs, metadata[1])
		modes = append(modes, metadata[0])
	}
	return objectIDs, modes, nil
}

func localEvidenceRefs(ctx context.Context, cwd string) ([]string, error) {
	result, err := inspectGit(ctx, cwd, nil, "for-each-ref", "--format=%(refname)", "refs/heads", "refs/tags", "refs/stash")
	if err != nil {
		return nil, err
	}
	text := strings.TrimSuffix(string(result.stdout), "\n")
	if text == "" {
		return nil, nil
	}
	refs := strings.Split(text, "\n")
	for _, ref := range refs {
		if ref == "" || strings.ContainsAny(ref, "\x00\r") {
			return nil, errors.New("invalid evidence ref")
		}
	}
	return refs, nil
}

func exactBytesReachableAtPath(ctx context.Context, cwd string, refs []string, casualty Casualty) (bool, error) {
	if len(refs) == 0 {
		return false, nil
	}
	argv := append([]string{"log", "-z", "--format=%H", "--no-renames"}, refs...)
	argv = append(argv, "--", casualty.path)
	result, err := inspectGit(ctx, cwd, nil, argv...)
	if err != nil {
		return false, err
	}
	parts := bytes.Split(result.stdout, []byte{0})
	if len(parts) == 0 || len(parts[len(parts)-1]) != 0 {
		return false, errors.New("invalid reachable commit output")
	}
	objectSpecs := make([]string, 0, len(parts)-1)
	for _, commit := range parts[:len(parts)-1] {
		if len(commit) == 0 {
			return false, errors.New("empty reachable commit id")
		}
		objectSpecs = append(objectSpecs, string(commit)+":"+casualty.path)
	}
	objects, err := inspectObjectSpecs(ctx, cwd, objectSpecs)
	if err != nil {
		return false, err
	}
	for _, object := range objects {
		if object.typ == "blob" && bytes.Equal(object.data, casualty.content) {
			return true, nil
		}
	}
	return false, nil
}
