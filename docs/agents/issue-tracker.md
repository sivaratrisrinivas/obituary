# Issue Tracker — GitHub Issues

Issues for this repo live in **GitHub Issues** at
<https://github.com/sivaratrisrinivas/obituary/issues>.

## CLI

All issue operations use the [`gh`](https://cli.github.com/) CLI.

## Creating issues

```bash
gh issue create --title "Title" --body "Body" --label "label"
```

## Listing issues

```bash
gh issue list --state open
```

## Closing issues

```bash
gh issue close <number>
```

## PRs as a request surface

Disabled. External PRs are **not** routed through the triage queue.
