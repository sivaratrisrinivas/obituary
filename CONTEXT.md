# Domain Context

## Product boundary

Obituary is an explicitly invoked, local preflight for four exact Git restore/checkout path-mode command forms. It analyzes repository state and reports exact worktree content at risk. It is not a command interceptor, recovery service, or security boundary.

## Glossary

### Proposed command

The exact argv following Obituary's `--` separator. Only `git restore .`, `git restore -- .`, `git checkout .`, and `git checkout -- .` are supported.

### Casualty

The exact current regular-file worktree bytes and compatible executable semantics that a supported command will overwrite from the index. A repository path by itself is not a casualty.

### Replacement

The index state Git will write to the worktree. Staged content is a replacement that survives the supported command; it is not the casualty when additional unstaged content exists.

### Exact same-path copy

Evidence whose bytes and executable semantics match the casualty at the identical repository-relative path and whose source is the index or a named reachable local branch, tag, or stash.

### Complete

The supported command and relevant repository semantics were fully analyzed, and every casualty has a finished exact-evidence result. A complete report may contain zero casualties.

### Unknown

A command or relevant repository, file, or transformation semantic is unsupported or cannot be resolved truthfully. Unknown is a whole-analysis result and carries no per-file safety verdicts.

### Search incomplete

The declared evidence search did not finish because of deadline or source failure. Search incomplete is not a negative evidence result.

### Oracle repository

The disposable repository in which the real supported Git command is executed. Its exact before/after state independently determines what Git overwrote.

## Core invariants

- Analysis is read-only.
- Evidence is exact, same-path, mode-compatible, and reachable through a declared local source.
- Untracked, ignored, staged-only, absent-worktree, and out-of-scope paths are not casualties.
- Unknown and incomplete outcomes never become partial or negative safety claims.
- Explicit proceed is required before `run` executes Git; cancellation is the default.
- Any false positive evidence claim or supported casualty mismatch blocks release.

Detailed product and engineering rules live in `.kiro/steering/`. The accepted feature contract lives in `.kiro/specs/restore-preflight/`.