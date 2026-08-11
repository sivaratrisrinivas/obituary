# Restore Preflight Design

## Overview

Obituary is a local Go CLI with one deep semantic module. The module receives a working directory and proposed argv, recognizes only four exact restore/checkout forms, inspects real Git state without mutation, identifies exact regular-file worktree casualties, searches bounded reachable local Git sources for byte- and mode-exact same-path evidence, and returns a result that cannot confuse completeness with uncertainty.

The CLI renders that result. `explain` stops there. `run` adds a controlling-terminal prompt and dispatches the exact proposed Git command only after an explicit proceed choice on a complete result.

## Architecture

```text
CLI: parse mode and `--`
        |
        v
Analyze(ctx, cwd, argv) -> Report
  | command recognition
  | repository/precondition inspection
  | casualty discovery
  | exact evidence search
  | deadline and invariant enforcement
        |
        v
Renderer -> terminal text
        |
        +-- explain: exit
        +-- run: TTY prompt -> cancel or exec Git
```

There is one primary testing seam: `Analyze`. Git command execution is an implementation detail behind private helpers and is tested with real disposable repositories rather than a public mock interface.

## Components and interfaces

### Command package

The command package:

- accepts only `explain`, `run`, help, and version behavior;
- requires `--` before the proposed Git argv;
- passes the current working directory and exact argv to `Analyze`;
- renders reports through centralized wording;
- maps complete and non-complete outcomes to documented exit codes;
- detects a controlling TTY and reads a single guarded choice;
- replaces its process with the resolved real Git executable after explicit proceed where practical.

It does not inspect repository state or construct safety verdicts independently.

### Analysis module

The internal analysis module exposes an interface close to:

```go
func Analyze(ctx context.Context, cwd string, argv []string) Report
```

It owns all Git semantics and returns one validated report variant.

Conceptual result model:

```text
Report
  CompleteReport
    command
    repository root
    casualties[]
  UnknownReport
    command if recognized
    named reason
  SearchIncompleteReport
    command
    named reason

Casualty
  repository-relative path as raw bytes or lossless path value
  current raw byte size
  current content identity
  executable semantics
  textual additions/deletions or binary marker
  Evidence

Evidence
  ExactSamePathCopyFound(locator)
  NoExactSamePathCopyFound(checked-scope)
```

Constructors or validation prevent these invalid states:

- `UNKNOWN` or `SEARCH_INCOMPLETE` with per-file safety verdicts;
- incomplete search with negative evidence;
- positive evidence without a concrete locator;
- complete casualty without exact current content identity;
- a path that cannot be rendered losslessly being silently altered.

## Data flow

### 1. Recognize exact argv

Compare argv as a sequence, not as reconstructed shell text. Only the four requirement-listed vectors continue. Every other sequence returns unsupported without executing Git.

### 2. Establish repository and pathspec context

Resolve the supplied working directory and repository root through non-mutating Git inspection. Confirm a normal non-bare worktree and derive the relative `.` scope from the invocation directory rather than assuming repository root.

### 3. Enforce whole-analysis preconditions

Before producing casualties, detect conditions that can invalidate completeness:

- unmerged index entries;
- sparse checkout;
- dirty affected submodules;
- potentially affected unsupported file types;
- relevant content filters;
- `working-tree-encoding`;
- unsafe EOL conversion or normalization;
- any unresolved repository/path semantics.

A relevant failure returns one `UNKNOWN` result. The analyzer does not classify a convenient subset.

### 4. Discover casualties

Use NUL-delimited Git output and raw filesystem reads to compare tracked worktree states under the relative `.` pathspec with index entries. A casualty exists only when current regular-file worktree content or executable semantics will be replaced by the index.

The discovery step excludes:

- untracked and ignored paths;
- tracked paths whose worktree already matches the index;
- tracked paths absent from the worktree;
- paths outside invocation scope.

For each casualty, compute an exact content identity from raw current bytes and retain compatible executable semantics. Text line deltas are display metadata, not identity; if Git cannot calculate them reliably, use a binary marker.

### 5. Search exact same-path evidence

Search the index and history reachable from local branch, tag, and stash refs. Evidence counts only when all of these match:

- repository-relative path;
- raw file bytes;
- supported regular-file type;
- compatible executable semantics;
- reachability through a concrete named local source.

Different bytes, a different path, an unreachable object, or the post-command index replacement do not establish preservation of the casualty.

Prefer batched object inspection and deduplicate equivalent object/path candidates. A concrete locator identifies the named source and path, for example a ref-qualified path. Search ordering may choose a deterministic preferred locator, but it must not broaden the claim.

### 6. Enforce deadline and classify

All inspection honors `context.Context`. If the declared search does not finish because of deadline or source failure, return `SEARCH_INCOMPLETE` for the entire analysis. Only a finished search can produce positive or bounded negative evidence for every casualty and a `COMPLETE` report.

### 7. Render

Rendering groups casualties by evidence result, preserves paths losslessly, and carries meaning through text rather than color. The bounded negative phrase and excluded-source disclosure are centralized constants or functions. ANSI color is optional only on terminals and is disabled by `NO_COLOR`.

## Git subprocess policy

Private helpers execute Git directly with argument arrays. Every parsed inspection command receives a controlled environment including `LC_ALL=C` and `GIT_OPTIONAL_LOCKS=0`. Commands request NUL-delimited or otherwise stable plumbing/porcelain output. External diff and text conversion are disabled where they could affect semantics.

Analysis must not invoke shell evaluation, refresh the index, acquire optional locks, write refs/configuration, contact remotes, or execute user-supplied hooks as part of inspection. Process count should be bounded through batch protocols where available.

## Mutation invariant

The oracle snapshots protected state before and after `Analyze`, including:

- tracked and relevant untracked worktree bytes and modes;
- exact index file bytes;
- local refs and stash refs;
- relevant repository and user-visible configuration;
- any other filesystem paths analysis could plausibly touch.

Any difference fails verification. Cancellation tests apply the same comparison around `run`.

## Error handling

Errors are semantic outcomes when possible:

- unrecognized argv: unsupported command;
- unsupported or unresolved repository state: `UNKNOWN` with one actionable reason;
- deadline or incomplete declared source: `SEARCH_INCOMPLETE` with one actionable reason;
- complete analysis: zero or more casualties with complete evidence results.

Diagnostics avoid implying safety after failure. `run` never offers normal proceed for unsupported, unknown, or incomplete outcomes; users who intentionally accept that risk may invoke Git directly.

## Testing strategy

### Paired real-Git oracle

Each declarative scenario generates two repositories independently from the same state description. Repository A is analyzed. Repository B executes the actual supported Git command. Exact before/after worktree bytes and modes in B derive the casualties Git overwrote. Tests compare that set with the complete report from A.

Evidence tests independently construct reachable commits, branches, tags, stashes, different historical bytes, and unreachable objects. They verify both positive locators and bounded negative outcomes.

### Scenario matrix

The release corpus covers approximately 18–25 adversarial scenarios:

1. unstaged text modification;
2. staged-only modification;
3. staged plus additional unstaged content;
4. tracked worktree deletion;
5. ordinary untracked file;
6. ignored file;
7. repository-root invocation;
8. nested working-directory invocation;
9. affected and unaffected sibling directories;
10. spaces in filenames;
11. Unicode filenames;
12. binary files;
13. executable-bit changes;
14. exact same-path reachable commit evidence;
15. same-path different-byte history;
16. exact same-path stash evidence;
17. unreachable exact object;
18. unmerged index;
19. sparse checkout;
20. symlink casualty;
21. dirty submodule;
22. active filter or EOL transformation;
23. newline in a path, handled losslessly or refused as `UNKNOWN`;
24. analysis deadline;
25. `run` cancellation.

### CLI tests

CLI-level tests cover parsing, output grouping, bounded wording, exit codes, non-TTY refusal, default cancellation, Escape/cancel behavior, and exact Git dispatch after explicit proceed. They complement rather than duplicate semantic oracle tests.

## Scope control

The implementation adds no transparent wrapper, installer, snapshot action, arbitrary pathspec, other destructive command family, shell parser, external-copy scan, cross-path duplicate search, reflog evidence, JSON schema, persistent configuration, telemetry, statistics, network behavior, account, or runtime model. A newly discovered semantic that cannot be proved is routed to `UNKNOWN` or removed from supported scope rather than approximated.