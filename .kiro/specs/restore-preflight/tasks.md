# Restore Preflight Implementation Plan

Each task is a small vertical slice. Work test-first: establish a failing observable behavior, implement only enough to pass it, run focused verification, and commit the green slice locally. Task 1 is the sole exception: it commits the validated red oracle that Task 2 turns green through the pre-agreed `Analyze` seam. Do not publish or push without separate authorization.

At the end of every slice, record:

- behavior added;
- Git semantics relied upon;
- safety invariant protected;
- test evidence;
- known unsupported states;
- files and local commit changed.

- [x] 1. Establish the Go module and paired repository oracle
  - Initialize the local Go module as `obituary` and add only the directories needed by the first test.
  - Build a declarative fixture capable of generating two independent disposable Git repositories with identical relevant state.
  - Capture exact worktree bytes/modes and index bytes before analysis; execute real Git only in the oracle repository.
  - Add the first failing scenario: one tracked text file with an unstaged modification under repository-root `git restore .`.
  - Verify that the test fails because `Analyze` behavior is absent, not because the fixture is invalid.
  - _Requirements: R2, R3, R12_
  - _Dependencies: none_
  - **Completion record (2026-08-12):**
    - Behavior added: initialized module `obituary`; added a declarative fixture that independently creates analyzed and oracle repositories and derives one overwritten worktree state from real Git.
    - Git semantics relied upon: repository-root `git restore .` replaces the tracked unstaged regular-file bytes with index bytes; in this fixture it preserves mode `0644`.
    - Safety invariant protected: the proposed destructive command runs only in the disposable oracle repository; the paired analyzed repository is not restored.
    - Test evidence: `go test -run '^$' ./internal/obituary` passes compilation; `go test -count=1 -v ./internal/obituary` validates the fixture and oracle, then fails only at the explicit absent-`Analyze` red sentinel.
    - Known unsupported states: all production analysis, report, evidence-search, repository-precondition, and transformation semantics remain unimplemented for task 2 and later.
    - Files and local commit changed: `go.mod` and `internal/obituary/analyze_test.go` in `960e6a1`; this task record documents the intentional red checkpoint before Task 2.

- [ ] 2. Recognize commands and pass the first complete casualty oracle
  - Define validated report variants and the `Analyze(ctx, cwd, argv)` seam.
  - Recognize only the four exact supported argv forms; add table tests for near misses.
  - Implement the minimum read-only worktree-versus-index inspection needed to identify the first casualty.
  - Complete the declared index and reachable local branch, tag, and stash search for the simple fixture so the public report can truthfully return bounded negative evidence.
  - Assert that analysis leaves worktree, index, and refs unchanged.
  - _Requirements: R1, R2, R3, R5, R6_
  - _Dependencies: 1_

- [ ] 3. Complete core casualty and non-casualty semantics
  - Run the paired real-Git oracle under all four supported argv forms, including both `--` variants.
  - Add staged-only, staged-plus-unstaged, tracked deletion, untracked, ignored, and zero-casualty scenarios one at a time.
  - Add repository-root, nested-working-directory, and unaffected-sibling coverage for the relative `.` pathspec.
  - Add spaces, Unicode, binary content, and executable semantics while preserving paths losslessly.
  - Route any path representation that cannot be handled truthfully to `UNKNOWN`.
  - _Requirements: R1, R2, R3, R4, R7, R12_
  - _Dependencies: 2_

- [ ] 4. Enforce repository and transformation preconditions
  - Add failing fixtures for unmerged index, sparse checkout, symlink casualty, dirty affected submodule, active filters, `working-tree-encoding`, and unsafe EOL behavior.
  - Implement whole-analysis precondition checks that return one named `UNKNOWN` outcome without partial casualty verdicts.
  - Hash protected repository state before and after every unknown-path analysis.
  - _Requirements: R2, R7, R12_
  - _Dependencies: 3_

- [ ] 5. Harden exact same-path reachable commit evidence
  - Add failing cases for exact current casualty bytes at the same path in a reachable local branch/tag and for older different bytes at that path.
  - Batch reachable object inspection and return a deterministic concrete locator for exact, mode-compatible evidence.
  - Prove that different bytes, different paths, post-command index state, and unreachable exact objects are not accepted.
  - _Requirements: R5, R12_
  - _Dependencies: 3, 4_

- [ ] 6. Add stash evidence and bounded negative results
  - Add exact same-path stash fixtures and return a concrete stash-qualified locator.
  - Harden the completed-search negative variant so it is constructed only after all declared sources have been inspected.
  - Centralize the required negative phrase and excluded-source disclosure.
  - _Requirements: R5, R6, R12_
  - _Dependencies: 5_

- [ ] 7. Add deadline and incomplete-search behavior
  - Add a deterministic test that interrupts analysis before the declared evidence search completes.
  - Return `SEARCH_INCOMPLETE` with a named reason and no casualty-level negative evidence.
  - Verify source inspection failures cannot collapse into complete negative results.
  - _Requirements: R8, R12_
  - _Dependencies: 6_

- [ ] 8. Implement the read-only `explain` CLI slice
  - Parse the Obituary mode and required `--` separator without shell reconstruction.
  - Render complete, unknown, and incomplete reports with lossless paths, terminal-aware color, and `NO_COLOR` support.
  - Implement exit 0 for complete reports and exit 2 with stderr diagnostics for unknown/incomplete reports.
  - Add CLI tests proving `explain` never prompts or executes Git.
  - _Requirements: R1, R6, R9, R11_
  - _Dependencies: 7_

- [ ] 9. Implement guarded `run` cancellation
  - Require a controlling TTY and block normal proceed for non-complete outcomes.
  - Render `[p] proceed` and `[enter/esc] cancel` only after a complete report.
  - Add Enter, Escape, and interruption tests that compare complete repository state before and after cancellation.
  - _Requirements: R2, R10, R11, R12_
  - _Dependencies: 8_

- [ ] 10. Execute exact Git argv after explicit proceed
  - Add a disposable-repository end-to-end test that selects proceed and observes real Git behavior.
  - Resolve and replace the process with the real Git executable where supported, preserving stdio, signals, and exit behavior.
  - Ensure no alias expansion, shell evaluation, bypass variable, or alternative argv is introduced.
  - _Requirements: R1, R10, R11_
  - _Dependencies: 9_

- [ ] 11. Complete adversarial verification and local automation
  - Finish the 18–25 scenario release matrix and protect index bytes, refs, worktree/filesystem state, and configuration around analysis.
  - Add focused tests, race/static checks where appropriate, and a `make verify` target.
  - Add a repository-local CI workflow that runs the documented verification without publishing, deploying, or requiring secrets.
  - Confirm zero false evidence, 100% supported casualty agreement, 100% unknown classification for declared unsupported states, zero analysis/cancellation mutation, and runtime under 60 seconds.
  - Tighten the Kiro save hook target if the final package layout differs from `internal/obituary`.
  - _Requirements: R2, R7, R8, R12_
  - _Dependencies: 10_

- [ ] 12. Document and review the release candidate
  - Replace the minimal README with build, test, supported-command, exact-evidence, limitation, bypass, offline, and Kiro-process documentation backed by reproducible commands.
  - Add decision and development records for the parsimony cuts and one authentic spec-to-oracle correction.
  - Run a standards-and-spec review, fresh-clone verification, and security audit of read-only inspection, process execution, TTY handling, quoting, and no-network claims.
  - Prepare the demo against the release commit; publication, push, release creation, and external submission remain separately authorized actions.
  - _Requirements: R1–R12_
  - _Dependencies: 11_

## Checkpoints

### After task 4 — casualty semantics

- [ ] All current focused tests pass.
- [ ] Supported core casualty scenarios agree with real Git.
- [ ] Every declared unsupported fixture returns whole-analysis `UNKNOWN`.
- [ ] Analysis mutation remains zero.

### After task 7 — evidence semantics

- [ ] Positive evidence has zero false claims and a concrete named locator.
- [ ] Different bytes, paths, and unreachable objects are rejected.
- [ ] Incomplete search cannot produce a negative claim.

### After task 10 — user workflow

- [ ] `explain` is noninteractive and read-only.
- [ ] `run` defaults to cancel and refuses non-TTY execution.
- [ ] Explicit proceed executes exactly the proposed Git argv in a disposable repository.

### After task 12 — release gate

- [ ] `make verify` passes within 60 seconds on the development machine.
- [ ] Documentation and demo claims are reproducible from the release commit.
- [ ] No excluded feature or external mutation has entered scope.
