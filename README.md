# Obituary

> Know the exact unstaged file states `git restore .` will overwrite before you decide to run it.

Obituary is a local, explicitly invoked Git preflight. For a deliberately narrow set of restore commands, it is designed to identify the exact regular-file worktree content that Git will replace from the index and determine whether those exact bytes still exist at the same path in bounded, reachable local Git history.

Obituary is not a transparent Git wrapper, a recovery service, or a security boundary. Direct Git invocation remains the explicit bypass.

> [!IMPORTANT]
> **Current status: Task 1 complete.** The Go module and first paired real-Git oracle are implemented. The oracle is an intentional red checkpoint: it validates Git's overwrite behavior, then fails at the explicit missing-`Analyze` sentinel that Task 2 will replace. The CLI and production analysis are not implemented yet, so the commands below describe the accepted interface rather than a runnable release.

## The problem

Consider a tracked file with three relevant states:

1. an older committed version;
2. staged content **B** in the index;
3. additional unstaged content **A** in the worktree.

Running `git restore .` replaces the worktree with the index. Content **B** survives, while the exact current worktree state containing **A** is overwritten.

A filename alone does not explain that loss. Saying the file is tracked does not prove the current bytes are preserved. Finding an older, different version at the same path does not recover **A**.

Obituary's core question is precise:

> Which exact worktree states will this command overwrite, and do those exact bytes with compatible executable semantics survive at the same path in the index or a reachable local branch, tag, or stash?

## Accepted command interface

The MVP supports only these proposed Git argument vectors:

```text
git restore .
git restore -- .
git checkout .
git checkout -- .
```

They are invoked through one of two Obituary modes:

```text
obituary explain -- git restore .
obituary run -- git restore .
```

### `explain`

The accepted design requires `explain` to:

- analyze without executing the proposed Git command;
- avoid changing filesystem content, worktree state, index bytes, refs, or configuration;
- return a complete report for supported, fully inspected states;
- remain usable without a terminal;
- exit 0 for a complete analysis, including zero casualties;
- exit 2 for `UNKNOWN` or `SEARCH_INCOMPLETE`.

### `run`

The accepted design requires `run` to perform the identical analysis, then:

- require a controlling terminal;
- offer proceed only after a complete result;
- treat Enter, Escape, or interruption as cancellation;
- execute the exact proposed Git argv only after explicit `p` confirmation;
- leave repository state unchanged on cancellation.

If Obituary cannot provide a complete verdict, users may intentionally bypass it by invoking Git directly.

## What counts as a casualty?

A **casualty** is the exact current regular-file worktree bytes and compatible executable semantics that the supported command will overwrite from the index. A path by itself is not a casualty.

The following are not casualties of the supported forms:

- ordinary untracked files;
- ignored files;
- staged-only changes whose worktree already matches the index;
- tracked paths currently absent from the worktree, which Git will recreate;
- modifications outside the relative `.` pathspec rooted at the invocation directory.

If a potentially affected path has unsupported semantics, Obituary must return one whole-analysis `UNKNOWN` result rather than classify a convenient subset.

## Exact evidence, not recovery theater

Positive evidence requires all of the following:

- byte-for-byte equality with the current casualty;
- compatible executable semantics;
- the identical repository-relative path;
- reachability through the index or a named local branch, tag, or stash;
- survival of the proposed restore operation.

Different historical bytes, identical bytes at a different path, and unreachable Git objects do not count.

A completed search with no evidence is deliberately bounded:

> **NO EXACT COPY FOUND AT THIS PATH IN THE INDEX OR REACHABLE LOCAL GIT REFS**
>
> Not checked: other paths, remotes, external backups, editor history, filesystem snapshots.

Obituary must never turn that bounded result into “gone forever” or an unqualified claim that no copy exists.

## Result model

Analysis has three distinct whole-report outcomes:

| Outcome | Meaning |
| --- | --- |
| `COMPLETE` | Supported semantics were fully inspected. The report may contain zero or more casualties, each with finished evidence. |
| `UNKNOWN` | A command, repository, file, or transformation semantic cannot be classified truthfully. No partial safety verdict is produced. |
| `SEARCH_INCOMPLETE` | The declared evidence search timed out or failed before completion. No negative evidence claim is produced. |

A complete casualty has one evidence result:

- `EXACT_SAME_PATH_COPY_FOUND`, with a concrete locator; or
- `NO_EXACT_SAME_PATH_COPY_FOUND`, with the bounded checked scope.

## Conservative repository support

The first release is designed for a normal, non-bare local repository with a valid index, regular-file casualties, no unmerged entries, no sparse checkout, no dirty affected submodules, and no relevant content transformation.

Relevant filters, `working-tree-encoding`, or unsafe EOL conversion produce `UNKNOWN`. The MVP will not emulate Git clean/smudge pipelines.

## Architecture

Obituary is a Go standard-library module backed by the installed Git executable as the semantic authority. The production CLI and analyzer remain planned work.

The primary seam is intentionally small:

```go
func Analyze(ctx context.Context, cwd string, argv []string) Report
```

One deep `internal/obituary` module will own command recognition, repository preconditions, Git plumbing, casualty identity, exact evidence, deadlines, and truthful result classification. The command layer will only parse the Obituary mode, call `Analyze`, render the report, prompt, and dispatch Git after explicit confirmation.

Git inspection must use direct argv execution, stable NUL-delimited formats, `LC_ALL=C`, `GIT_OPTIONAL_LOCKS=0`, and read-only plumbing. The first test already builds paired disposable repositories and uses real Git for its oracle; Task 2 will connect that oracle to `Analyze` rather than introduce a mocked Git adapter.

See the accepted design in [`.kiro/specs/restore-preflight/design.md`](.kiro/specs/restore-preflight/design.md).

## Current verification

Task 1 requires Go 1.26 and the installed Git executable:

```sh
# Proves the package compiles while the intentional red test is excluded.
go test -run '^$' ./internal/obituary

# Validates the fixture and real-Git oracle, then fails only at the
# explicit missing-Analyze sentinel reserved for Task 2.
go test -count=1 -v ./internal/obituary
```

The full test command is expected to remain red until Task 2 implements the pre-agreed `Analyze` seam. Task 1 is successful only when its log reaches `fixture and real-Git oracle validated` before that sentinel.

## Verification strategy

The paired-repository oracle established in Task 1 generates two disposable repositories from the same declarative fixture. As subsequent tasks connect and expand it, the release flow will:

1. analyze repository A with Obituary;
2. execute the real supported Git command in repository B;
3. derive overwritten states from exact before/after bytes and modes in B;
4. compare them with Obituary's casualties;
5. independently construct and remove exact same-path evidence in refs and stashes.

Release gates are absolute:

- zero false exact-copy claims;
- 100% casualty agreement for supported scenarios;
- 100% `UNKNOWN` classification for declared unsupported states;
- zero analysis mutation;
- zero cancellation mutation;
- `make verify` under 60 seconds on the development machine.

The adversarial matrix and ordered implementation slices are in [`.kiro/specs/restore-preflight/tasks.md`](.kiro/specs/restore-preflight/tasks.md).

## Deliberate non-goals

The MVP does not include:

- transparent Git interception or shell wrappers;
- installers or shell configuration edits;
- snapshots or automatic recovery actions;
- `git reset`, `git clean`, `rm`, or other destructive command families;
- arbitrary pathspecs or general shell parsing;
- cross-path duplicate searches or reflog evidence;
- remote, backup, editor-history, or filesystem-snapshot searches;
- JSON output, persistent configuration, statistics, or a ledger;
- network access, telemetry, accounts, API keys, services, or runtime models.

Freed scope is reserved for semantic correctness, adversarial verification, documentation, and demonstration—not replacement features.

## Kiro-first development

The repository treats the Kiro artifacts as executable project history rather than generated decoration:

- [product steering](.kiro/steering/product.md) defines the user promise and bounded language;
- [technology steering](.kiro/steering/tech.md) defines Git inspection and verification discipline;
- [structure steering](.kiro/steering/structure.md) protects the single deep module;
- [safety invariants](.kiro/steering/safety-invariants.md) establish release-blocking guarantees;
- [requirements](.kiro/specs/restore-preflight/requirements.md), [design](.kiro/specs/restore-preflight/design.md), and [tasks](.kiro/specs/restore-preflight/tasks.md) form the accepted `restore-preflight` spec;
- [the save hook](.kiro/hooks/test-restore-preflight-on-save.json) runs the focused semantic package test when `internal/obituary` Go files are saved; it shares Task 1's intentional red state until Task 2 implements `Analyze`.

Each implementation slice must record the behavior added, Git semantics relied upon, protected invariant, test evidence, known unsupported states, and files and local commit changed.

## Repository map

```text
.kiro/
  hooks/                         focused test automation
  specs/restore-preflight/       accepted requirements, design, and tasks
  steering/                      product, technology, structure, and safety rules
CONTEXT.md                       domain glossary and core invariants
docs/agents/                     agent workflow conventions
docs/adr/                        architecture decision records
go.mod                           Go module definition
internal/obituary/analyze_test.go first paired real-Git oracle
```

The remaining planned product layout is documented in [`.kiro/steering/structure.md`](.kiro/steering/structure.md). Production analyzer and command files do not exist yet.

## Development status

- [x] Parsimonious product boundary accepted
- [x] Domain language and safety invariants recorded
- [x] Requirements, design, and incremental task plan committed
- [x] Focused Kiro test hook created
- [x] Go module initialized
- [x] First paired real-Git oracle validated at its intentional red checkpoint
- [ ] `Analyze` implementation started
- [ ] CLI implemented
- [ ] Adversarial release gates passing
- [ ] Reproducible build and release available

Until the remaining implementation items are complete, there is no installation command or working binary to advertise.
