package obituary

import (
	"errors"
	"slices"
)

// Outcome identifies the whole-analysis result without weakening its concrete variant.
type Outcome string

const (
	Complete         Outcome = "COMPLETE"
	Unknown          Outcome = "UNKNOWN"
	SearchIncomplete Outcome = "SEARCH_INCOMPLETE"
)

// Reason names why analysis could not produce a complete report.
type Reason string

const (
	UnsupportedCommand Reason = "unsupported command"
	UnsupportedState   Reason = "unsupported repository state"
	InspectionFailed   Reason = "repository inspection failed"
	SearchInterrupted  Reason = "evidence search interrupted"
)

const (
	NoExactCopyClaim = "NO EXACT COPY FOUND AT THIS PATH IN THE INDEX OR REACHABLE LOCAL GIT REFS"
	NotCheckedClaim  = "Not checked: other paths, remotes, external backups, editor history, filesystem snapshots."
)

// Report is one validated, structurally distinct whole-analysis result.
type Report interface {
	Outcome() Outcome
	isReport()
}

// CompleteReport contains only casualties with finished evidence searches.
type CompleteReport interface {
	Report
	Command() []string
	RepositoryRoot() string
	Casualties() []Casualty
	isCompleteReport()
}

type completeReport struct {
	command        []string
	repositoryRoot string
	casualties     []Casualty
}

func (completeReport) Outcome() Outcome         { return Complete }
func (completeReport) isReport()                {}
func (completeReport) isCompleteReport()        {}
func (r completeReport) Command() []string      { return slices.Clone(r.command) }
func (r completeReport) RepositoryRoot() string { return r.repositoryRoot }
func (r completeReport) Casualties() []Casualty { return slices.Clone(r.casualties) }

// UnknownReport contains no per-casualty verdicts.
type UnknownReport interface {
	Report
	Command() []string
	Reason() Reason
	isUnknownReport()
}

type unknownReport struct {
	command []string
	reason  Reason
}

func (unknownReport) Outcome() Outcome    { return Unknown }
func (unknownReport) isReport()           {}
func (unknownReport) isUnknownReport()    {}
func (r unknownReport) Command() []string { return slices.Clone(r.command) }
func (r unknownReport) Reason() Reason    { return r.reason }

// SearchIncompleteReport contains no casualty evidence or negative claim.
type SearchIncompleteReport interface {
	Report
	Command() []string
	Reason() Reason
	isSearchIncompleteReport()
}

type searchIncompleteReport struct {
	command []string
	reason  Reason
}

func (searchIncompleteReport) Outcome() Outcome          { return SearchIncomplete }
func (searchIncompleteReport) isReport()                 {}
func (searchIncompleteReport) isSearchIncompleteReport() {}
func (r searchIncompleteReport) Command() []string       { return slices.Clone(r.command) }
func (r searchIncompleteReport) Reason() Reason          { return r.reason }

// Casualty is the exact current worktree content and executable semantics at risk.
type Casualty struct {
	path       string
	content    []byte
	executable bool
	delta      Delta
	evidence   Evidence
}

func (c Casualty) Path() string       { return c.path }
func (c Casualty) Content() []byte    { return slices.Clone(c.content) }
func (c Casualty) Executable() bool   { return c.executable }
func (c Casualty) Delta() Delta       { return c.delta }
func (c Casualty) Evidence() Evidence { return c.evidence }

// Delta is either a textual line count or an explicit binary marker.
type Delta interface {
	isDelta()
}

// TextDelta reports Git's additions and deletions for a casualty.
type TextDelta interface {
	Delta
	Additions() int
	Deletions() int
	isTextDelta()
}

type textDelta struct {
	additions int
	deletions int
}

func (textDelta) isDelta()         {}
func (textDelta) isTextDelta()     {}
func (d textDelta) Additions() int { return d.additions }
func (d textDelta) Deletions() int { return d.deletions }

// BinaryDelta marks a casualty whose line delta is not reliably textual.
type BinaryDelta interface {
	Delta
	isBinaryDelta()
}

type binaryDelta struct{}

func (binaryDelta) isDelta()       {}
func (binaryDelta) isBinaryDelta() {}

// Evidence is a finished same-path evidence result.
type Evidence interface {
	isEvidence()
}

// ExactSamePathCopyFound identifies a concrete named local source.
type ExactSamePathCopyFound interface {
	Evidence
	Locator() string
	isExactSamePathCopyFound()
}

type exactSamePathCopyFound struct {
	locator string
}

func (exactSamePathCopyFound) isEvidence()               {}
func (exactSamePathCopyFound) isExactSamePathCopyFound() {}
func (e exactSamePathCopyFound) Locator() string         { return e.locator }

// NoExactSamePathCopyFound is valid only after every declared source was searched.
type NoExactSamePathCopyFound interface {
	Evidence
	Claim() string
	NotChecked() string
	isNoExactSamePathCopyFound()
}

type noExactSamePathCopyFound struct{}

func (noExactSamePathCopyFound) isEvidence()                 {}
func (noExactSamePathCopyFound) isNoExactSamePathCopyFound() {}
func (noExactSamePathCopyFound) Claim() string               { return NoExactCopyClaim }
func (noExactSamePathCopyFound) NotChecked() string          { return NotCheckedClaim }

func newCompleteReport(command []string, repositoryRoot string, casualties []Casualty) (CompleteReport, error) {
	if repositoryRoot == "" {
		return nil, errors.New("complete report requires repository root")
	}
	for _, casualty := range casualties {
		if casualty.path == "" || casualty.content == nil || casualty.delta == nil || casualty.evidence == nil {
			return nil, errors.New("complete casualty requires path, content, delta, and evidence")
		}
		if found, ok := casualty.evidence.(ExactSamePathCopyFound); ok && found.Locator() == "" {
			return nil, errors.New("positive evidence requires locator")
		}
	}
	return completeReport{
		command:        slices.Clone(command),
		repositoryRoot: repositoryRoot,
		casualties:     slices.Clone(casualties),
	}, nil
}

func newUnknownReport(command []string, reason Reason) UnknownReport {
	if reason == "" {
		panic("unknown report requires reason")
	}
	return unknownReport{command: slices.Clone(command), reason: reason}
}

func newSearchIncompleteReport(command []string, reason Reason) SearchIncompleteReport {
	if reason == "" {
		panic("search-incomplete report requires reason")
	}
	return searchIncompleteReport{command: slices.Clone(command), reason: reason}
}
