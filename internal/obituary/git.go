package obituary

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type gitOutput struct {
	stdout []byte
	stderr []byte
}

func inspectGit(ctx context.Context, cwd string, stdin []byte, argv ...string) (gitOutput, error) {
	cmd := exec.CommandContext(ctx, "git", argv...)
	cmd.Dir = cwd
	cmd.Env = inspectionEnvironment()
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return gitOutput{stdout: stdout.Bytes(), stderr: stderr.Bytes()}, fmt.Errorf("git %q: %w", argv, err)
	}
	return gitOutput{stdout: stdout.Bytes(), stderr: stderr.Bytes()}, nil
}

func inspectionEnvironment() []string {
	env := os.Environ()
	env = setEnvironment(env, "LC_ALL", "C")
	env = setEnvironment(env, "GIT_OPTIONAL_LOCKS", "0")
	env = setEnvironment(env, "GIT_NO_LAZY_FETCH", "1")
	env = setEnvironment(env, "GIT_TERMINAL_PROMPT", "0")
	return env
}

func setEnvironment(env []string, name, value string) []string {
	prefix := name + "="
	filtered := env[:0]
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return append(filtered, prefix+value)
}

type gitObject struct {
	oid  string
	typ  string
	data []byte
}

func inspectObjects(ctx context.Context, cwd string, objectIDs []string) (map[string]gitObject, error) {
	ordered, err := inspectObjectSpecs(ctx, cwd, objectIDs)
	if err != nil {
		return nil, err
	}
	objects := make(map[string]gitObject, len(ordered))
	for _, object := range ordered {
		objects[object.oid] = object
	}
	return objects, nil
}

func inspectObjectSpecs(ctx context.Context, cwd string, objectSpecs []string) ([]gitObject, error) {
	if len(objectSpecs) == 0 {
		return nil, nil
	}
	input := []byte(strings.Join(objectSpecs, "\n") + "\n")
	result, err := inspectGit(ctx, cwd, input, "cat-file", "--batch")
	if err != nil {
		return nil, err
	}

	objects := make([]gitObject, 0, len(objectSpecs))
	output := result.stdout
	for range objectSpecs {
		lineEnd := bytes.IndexByte(output, '\n')
		if lineEnd < 0 {
			return nil, errors.New("truncated cat-file header")
		}
		header := strings.Fields(string(output[:lineEnd]))
		output = output[lineEnd+1:]
		if len(header) != 3 {
			return nil, fmt.Errorf("unexpected cat-file header %q", header)
		}
		size, err := strconv.Atoi(header[2])
		if err != nil || size < 0 || len(output) < size+1 || output[size] != '\n' {
			return nil, errors.New("invalid cat-file object size")
		}
		objects = append(objects, gitObject{oid: header[0], typ: header[1], data: append([]byte(nil), output[:size]...)})
		output = output[size+1:]
	}
	if len(output) != 0 {
		return nil, errors.New("unexpected trailing cat-file output")
	}
	return objects, nil
}
