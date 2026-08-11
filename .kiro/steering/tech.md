# Technology

## Runtime

- Implement in Go using the standard library.
- Use the installed Git CLI as the authority for Git semantics.
- Operate locally and offline; runtime behavior requires no service, model, account, API key, or network request.
- Keep the command layer thin and place semantic analysis in one internal package.

## Git inspection discipline

All inspection subprocesses must:

- receive argument arrays directly rather than shell command strings;
- set `LC_ALL=C` for parsed output;
- set `GIT_OPTIONAL_LOCKS=0`;
- disable external diff and text conversion where relevant;
- request stable, NUL-delimited output for paths;
- avoid index refreshes, lock acquisition, configuration writes, ref changes, and worktree changes;
- batch object and history inspection where Git supports it instead of spawning once per file.

Raw worktree bytes are compared with Git object bytes only after relevant attributes and configuration establish that content transformation cannot invalidate the comparison. Active filters, `working-tree-encoding`, or unsafe EOL behavior produce `UNKNOWN`; the MVP does not emulate conversion pipelines.

## Result model

The analysis interface remains close to:

```go
func Analyze(ctx context.Context, cwd string, argv []string) Report
```

`Report` must make these overall outcomes structurally distinct:

- `COMPLETE`
- `UNKNOWN`
- `SEARCH_INCOMPLETE`

A complete report may contain zero or more casualties. Each casualty has one evidence result:

- `EXACT_SAME_PATH_COPY_FOUND`
- `NO_EXACT_SAME_PATH_COPY_FOUND`

An incomplete search never carries a negative evidence result. Invalid combinations should be unrepresentable or rejected by constructors/validation.

## Command behavior

- `explain` returns 0 for every complete analysis, including zero casualties.
- `explain` returns 2 for `UNKNOWN` or `SEARCH_INCOMPLETE` and writes diagnostics to stderr.
- `run` requires a controlling TTY.
- `run` offers proceed only after `COMPLETE`.
- Enter, Escape, or interruption before confirmation cancels without executing Git.
- On explicit proceed, replace the Obituary process with the real Git process where practical so Git owns terminal behavior and exit status.
- Suppress ANSI color for non-terminal output and when `NO_COLOR` is set.

## Verification

Tests exercise the exported analysis behavior against paired disposable repositories built from the same declarative fixture. One repository is analyzed; the other executes real Git. The oracle derives overwritten worktree states from exact before/after bytes and modes, then compares them with the report.

Release gates are absolute:

- zero false exact-copy claims;
- 100% casualty agreement for supported scenarios;
- 100% `UNKNOWN` classification for declared unsupported states;
- no filesystem, worktree, index, ref, or configuration mutation from `explain`;
- no mutation after cancellation;
- `make verify` completes in under 60 seconds on the development machine.

Prefer deterministic tests, `go test`, the race detector where useful, static analysis, and real Git fixtures over mocks of Git semantics.