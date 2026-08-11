# Safety Invariants

These invariants govern every implementation and documentation decision.

1. Analysis is read-only: it changes no filesystem content, worktree state, index bytes, refs, Git configuration, or user configuration.
2. Only the four exact proposed command forms in `product.md` can produce `COMPLETE`.
3. Unsupported command, repository, affected-file, or transformation semantics produce `UNKNOWN` for the entire analysis.
4. An unfinished declared search produces `SEARCH_INCOMPLETE`, never a negative exact-copy claim.
5. The casualty is the current worktree content and compatible executable semantics that the index replacement will overwrite; a path alone is not a casualty.
6. Older different bytes at the same path are not evidence for the current casualty.
7. Positive evidence is same-path, byte-exact, mode-compatible, and reachable through a named local branch, tag, or stash.
8. Untracked and ignored files are not casualties of the supported restore forms.
9. A tracked path currently absent from the worktree has no content casualty when Git recreates it.
10. `explain` never executes the proposed command.
11. `run` executes only after `COMPLETE`, a controlling TTY, and explicit proceed selection.
12. Enter, Escape, and interruption before proceed cancel without repository mutation.
13. Runtime behavior performs no network access, telemetry, update check, account operation, API-key access, configuration write, or persistent-state write.
14. User-facing claims remain bounded to searched sources and never say content is universally unrecoverable, gone forever, or edited for an inferred duration.
15. Every supported semantic rule has a paired real-Git oracle scenario; any false exact-copy result or supported casualty mismatch blocks release.