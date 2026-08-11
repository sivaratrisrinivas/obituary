# Domain Docs

## Layout

**Single-context** — this is not a monorepo.

- `CONTEXT.md` at the repo root holds the domain model (glossary, bounded contexts, key invariants).
- `docs/adr/` holds Architecture Decision Records (ADRs).

## Consumer rules

1. Before making a change that touches domain terminology, read `CONTEXT.md`.
2. Before proposing a new architectural decision, check `docs/adr/` for prior art.
3. When a skill writes a new ADR, place it at `docs/adr/NNNN-<slug>.md` (zero-padded, next sequential number).
4. Never modify an accepted ADR — supersede it with a new one that links back.
