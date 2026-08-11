# Restore Preflight Requirements

## Purpose

Obituary must tell a user exactly which tracked regular-file worktree states a narrowly supported Git restore command will overwrite from the index, and whether those exact bytes and compatible executable semantics survive at the same path in bounded, reachable local Git sources. The system favors an explicit unknown result over an unsafe or partial verdict.

## R1 — Exact command recognition

### User story

As a Git user, I want Obituary to analyze only command forms whose semantics it can prove, so that a familiar-looking but different command never receives a misleading verdict.

### Acceptance criteria

1. WHEN the proposed argv is exactly `git restore .`, `git restore -- .`, `git checkout .`, or `git checkout -- .`, THE SYSTEM SHALL attempt restore-preflight analysis.
2. WHEN the proposed argv differs in executable, subcommand, flags, argument count, pathspec, tree-ish, or ordering, THE SYSTEM SHALL report an unsupported command and SHALL NOT execute it.
3. WHEN command recognition fails, THE SYSTEM SHALL NOT inspect the result as though it were one of the supported forms.

## R2 — Read-only analysis

### User story

As a cautious Git user, I want explanation itself to be non-destructive, so that asking what will happen cannot alter the answer or my repository.

### Acceptance criteria

1. WHEN `explain` analyzes a command, THE SYSTEM SHALL leave filesystem content, worktree state, index bytes, refs, repository configuration, and user configuration unchanged.
2. WHEN Git inspection subprocesses run, THE SYSTEM SHALL suppress optional locking and avoid index refresh or write behavior.
3. WHEN analysis returns any outcome, THE SYSTEM SHALL NOT have executed the proposed destructive command.

## R3 — Exact casualty identity

### User story

As a Git user with unstaged work, I want the report to identify the precise content state at risk, so that staged and unstaged versions are never conflated.

### Acceptance criteria

1. WHEN a supported command will replace current regular-file worktree bytes from the index, THE SYSTEM SHALL report that current worktree state as one casualty.
2. WHEN a file contains staged content plus additional unstaged content, THE SYSTEM SHALL identify only the current worktree state as the casualty and SHALL treat the staged index state as the replacement that survives.
3. WHEN executable semantics affect exact restoration, THE SYSTEM SHALL include compatible executable semantics in casualty identity.
4. WHEN Git can reliably calculate textual line changes, THE SYSTEM SHALL report additions and deletions; OTHERWISE THE SYSTEM SHALL classify the delta as binary without weakening content identity.

## R4 — Non-casualties and relative scope

### User story

As a Git user, I want unaffected files excluded, so that the report is semantically precise rather than alarming.

### Acceptance criteria

1. WHEN a file is ordinary untracked or ignored, THE SYSTEM SHALL NOT report it as a casualty.
2. WHEN a tracked file has staged changes but its worktree matches the index, THE SYSTEM SHALL NOT report it as a casualty.
3. WHEN a tracked file is currently absent from the worktree and the command will recreate it, THE SYSTEM SHALL NOT report absence as destroyed content.
4. WHEN a changed path is outside the relative `.` pathspec rooted at the invocation working directory, THE SYSTEM SHALL NOT report it as a casualty.
5. WHEN no current worktree content will be overwritten, THE SYSTEM SHALL return a complete report with zero casualties.

## R5 — Exact same-path evidence

### User story

As a Git user, I want concrete evidence that the exact state survives, so that an older different version is not mistaken for recovery of my current work.

### Acceptance criteria

1. WHEN casualty bytes and compatible executable semantics occur at the same repository-relative path in the index or history reachable through a local branch, tag, or stash, THE SYSTEM SHALL report `EXACT_SAME_PATH_COPY_FOUND` with a concrete named locator.
2. WHEN content at the same path has different bytes, THE SYSTEM SHALL NOT count it as evidence.
3. WHEN exact bytes occur only at a different path, THE SYSTEM SHALL NOT count them as evidence.
4. WHEN exact bytes exist only in an unreachable object, THE SYSTEM SHALL NOT count them as evidence.
5. WHEN the index contains the post-command replacement but not the current casualty state, THE SYSTEM SHALL NOT describe the casualty as preserved merely because the path is tracked.

## R6 — Bounded negative claim

### User story

As a Git user, I want negative results to describe only what was actually searched, so that absence of local evidence is not presented as universal loss.

### Acceptance criteria

1. WHEN the complete declared search finds no exact evidence, THE SYSTEM SHALL report: `NO EXACT COPY FOUND AT THIS PATH IN THE INDEX OR REACHABLE LOCAL GIT REFS`.
2. WHEN it emits that negative result, THE SYSTEM SHALL state that other paths, remotes, external backups, editor history, and filesystem snapshots were not checked.
3. WHEN rendering any result, THE SYSTEM SHALL NOT say that content is gone forever, universally unrecoverable, or absent from an unsearched source.
4. WHEN rendering casualty metadata, THE SYSTEM SHALL NOT infer editing duration from modification time.

## R7 — Unsupported repository and file semantics

### User story

As a Git user in a complex repository, I want Obituary to admit uncertainty, so that unsupported semantics never become a partial safety verdict.

### Acceptance criteria

1. IF the working directory cannot be resolved inside a normal non-bare worktree, THEN THE SYSTEM SHALL return `UNKNOWN` with a named reason.
2. IF the index has unmerged entries or sparse checkout is active, THEN THE SYSTEM SHALL return `UNKNOWN`.
3. IF a potentially affected entry is not a supported regular file, THEN THE SYSTEM SHALL return `UNKNOWN` rather than omit that entry and classify the remainder.
4. IF an affected submodule is dirty, THEN THE SYSTEM SHALL return `UNKNOWN`.
5. IF relevant attributes or configuration enable a content filter, `working-tree-encoding`, or unsafe EOL transformation, THEN THE SYSTEM SHALL return `UNKNOWN`.
6. WHEN any relevant semantic cannot be resolved truthfully, THE SYSTEM SHALL return one whole-analysis `UNKNOWN` result without per-file safety verdicts.

## R8 — Incomplete search

### User story

As a Git user, I want a timed-out or failed search distinguished from a completed negative search, so that missing evidence is never treated as evidence of absence.

### Acceptance criteria

1. IF the analysis deadline expires before all declared evidence sources are inspected, THEN THE SYSTEM SHALL return `SEARCH_INCOMPLETE`.
2. IF a declared evidence source cannot be fully inspected, THEN THE SYSTEM SHALL return `SEARCH_INCOMPLETE` with a named reason.
3. WHEN the outcome is `SEARCH_INCOMPLETE`, THE SYSTEM SHALL NOT emit a no-copy claim or complete per-file evidence verdicts.

## R9 — Explain command behavior

### User story

As a user or automation process, I want a noninteractive explanation mode with stable exit behavior, so that analysis can be inspected without granting execution authority.

### Acceptance criteria

1. WHEN `obituary explain -- <supported-command>` completes, THE SYSTEM SHALL print the report and exit 0, including when there are zero casualties.
2. WHEN analysis returns `UNKNOWN` or `SEARCH_INCOMPLETE`, THE SYSTEM SHALL print a diagnostic to stderr and exit 2.
3. WHEN `explain` runs without a controlling terminal, THE SYSTEM SHALL remain usable and SHALL NOT prompt.

## R10 — Guarded execution and default cancellation

### User story

As a Git user, I want destructive execution to require an explicit, informed choice, so that proceeding cannot happen by accident.

### Acceptance criteria

1. WHEN `obituary run -- <supported-command>` has no controlling TTY, THE SYSTEM SHALL refuse to run the command.
2. WHEN analysis returns `UNKNOWN` or `SEARCH_INCOMPLETE`, THE SYSTEM SHALL block execution and SHALL NOT offer the normal proceed option.
3. WHEN analysis returns `COMPLETE`, THE SYSTEM SHALL show the report and offer `[p] proceed` and `[enter/esc] cancel`.
4. WHEN the user presses Enter, Escape, or interrupts before proceeding, THE SYSTEM SHALL cancel without executing Git and SHALL leave repository state unchanged.
5. WHEN the user explicitly selects proceed after a complete report, THE SYSTEM SHALL execute exactly the proposed Git argv and preserve Git's terminal behavior and exit result where the platform permits process replacement.

## R11 — Offline, stateless operation

### User story

As a user running a local safety tool, I want it to work without external dependencies or observation, so that repository analysis remains private and reproducible.

### Acceptance criteria

1. THE SYSTEM SHALL perform no network request during analysis, rendering, prompting, or execution dispatch.
2. THE SYSTEM SHALL require no account, API key, service, runtime model, configuration, telemetry, update check, or persistent application state.
3. THE SYSTEM SHALL NOT write user or Git configuration.

## R12 — Verification and release gate

### User story

As a maintainer, I want supported claims checked against real Git, so that release confidence comes from semantic agreement rather than mocks.

### Acceptance criteria

1. WHEN a supported oracle scenario runs, THE SYSTEM SHALL compare `Analyze` output with casualties independently derived by executing real Git in a paired disposable repository generated from the same declarative fixture.
2. WHEN testing evidence, THE SYSTEM SHALL independently add and remove exact same-path versions in reachable refs, stashes, and unreachable objects.
3. IF any supported casualty differs from the oracle, any positive evidence claim is false, any declared unsupported scenario avoids `UNKNOWN`, analysis mutates protected state, or cancellation mutates repository state, THEN verification SHALL fail.
4. WHEN `make verify` runs on the development machine, THE SYSTEM SHALL complete the release corpus in under 60 seconds.

## Scope boundary

This specification excludes transparent interception, shell wrappers, installers, snapshots, arbitrary pathspecs, `reset`, `clean`, `rm`, general shell parsing, external backup scans, cross-path duplicate searches, reflog evidence, JSON output, persistent configuration, statistics, telemetry, accounts, runtime models, network remotes, and every command form not listed in R1.