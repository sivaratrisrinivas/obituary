# Structure

## Repository layout

Keep the implementation small and organized around one deep semantic module:

```text
cmd/obituary/main.go
internal/obituary/analyze.go
internal/obituary/git.go
internal/obituary/report.go
internal/obituary/analyze_test.go
internal/testrepo/              only when shared fixture behavior earns a package
```

Repository process artifacts live under:

```text
.kiro/steering/
.kiro/specs/restore-preflight/
.kiro/hooks/
```

Do not create packages merely to mirror domain nouns. Add a package only when it hides meaningful complexity behind a smaller interface.

## Module boundary

`internal/obituary` owns:

- exact command recognition;
- repository and pathspec preconditions;
- Git inspection subprocesses;
- casualty discovery and identity;
- content-transformation refusal;
- same-path reachable evidence search;
- analysis deadlines;
- complete, unknown, and incomplete result construction;
- report data needed for truthful rendering.

The command package owns only:

- parsing `explain`, `run`, help, and version;
- separating Obituary arguments from the proposed Git argv at `--`;
- calling `Analyze`;
- rendering the returned report;
- controlling-terminal confirmation;
- executing Git after explicit proceed.

The command package must not independently infer Git semantics or weaken an analysis result.

## Seams

The primary behavioral seam is `Analyze(ctx, cwd, argv) -> Report`. Tests call this seam against real disposable repositories and assert observable reports. Internal command execution helpers remain private; do not export a Git runner solely for mocking.

A second end-to-end seam may exercise the compiled CLI for exit codes, rendering, TTY refusal, cancellation, and execution. It must not replace the semantic oracle around `Analyze`.

## Locality

Keep command recognition, repository refusal rules, casualty construction, evidence classification, and result validation near the data they govern. Keep renderer wording centralized so bounded negative claims cannot drift across call sites.

Delete abstractions that merely move complexity into callers. Prefer a few cohesive files in `internal/obituary` over a broad tree of shallow packages.