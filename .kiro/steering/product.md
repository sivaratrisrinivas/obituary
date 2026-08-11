# Product

## Thesis

Obituary is an explicitly invoked preflight for one destructive Git operation. It shows the exact current tracked worktree states that a supported restore command will overwrite from the index, then reports whether those exact bytes and compatible executable semantics survive at the same path in bounded local Git history.

The product proves this claim against real repository state. It is not a transparent command interceptor or a security boundary.

## Supported workflow

Users invoke one of two commands:

```text
obituary explain -- git restore .
obituary run -- git restore .
```

`explain` performs read-only analysis and never executes the proposed Git command. `run` performs the same analysis, displays the same report, defaults to cancellation, and executes Git only after a complete result and explicit confirmation.

Only these proposed argument vectors are supported:

```text
git restore .
git restore -- .
git checkout .
git checkout -- .
```

They restore tracked paths under the current directory's relative `.` pathspec from the index.

## Product promises

- Identify the exact current regular-file worktree state that Git will overwrite, not merely its path.
- Distinguish staged content that survives in the index from additional unstaged content that will be lost.
- Exclude ordinary untracked files, ignored files, staged-only changes, tracked files currently absent from the worktree, and paths outside the relative `.` pathspec.
- Count evidence only when bytes match exactly, executable semantics are compatible, the path is identical, and the source is reachable through a named local branch, tag, or stash.
- Return `UNKNOWN` instead of a partial verdict whenever repository or file semantics prevent a complete answer.
- Return `SEARCH_INCOMPLETE` instead of a negative evidence claim whenever the declared search does not finish.
- Keep analysis offline, read-only, and free of persistent state.

## Bounded language

A completed search without evidence must say:

> NO EXACT COPY FOUND AT THIS PATH IN THE INDEX OR REACHABLE LOCAL GIT REFS

It must immediately disclose:

> Not checked: other paths, remotes, external backups, editor history, filesystem snapshots.

Never describe content as universally unrecoverable, gone forever, or absent from sources that were not searched. Never infer editing duration from filesystem metadata.

## Deliberate limits

The first release has no wrapper, installer, snapshot operation, passthrough mode, network access, telemetry, account, API key, runtime model, configuration, ledger, statistics, JSON API, or external-copy scan. It does not support arbitrary pathspecs, destructive command families beyond the four forms above, sparse checkout, unmerged indexes, transformed worktree content, or unsupported file types.

Direct Git invocation is the explicit bypass for users who accept risks Obituary cannot classify.