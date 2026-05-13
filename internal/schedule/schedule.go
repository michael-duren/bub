// Package schedule turns a run mode (manual one-shot or automatic Pomodoro
// loop) into a stream of timed steps for the TUI to count down.
package schedule

import (
	"time"

	"github.com/michael-duren/bub/internal/config"
)

// Kind distinguishes work blocks from breaks.
type Kind int

const (
	// Work is a focused work block.
	Work Kind = iota
	// ShortBreak is the regular break between work blocks.
	ShortBreak
	// LongBreak is the longer break after a full set of work blocks.
	LongBreak
)

// Step is a single timed segment of a run.
type Step struct {
	Kind     Kind
	Duration time.Duration

	// Ordinal is the 1-based sequence number of this step among steps of the
	// same family — work blocks are numbered separately from breaks. It is 1
	// in a manual one-shot run.
	Ordinal int

	// SetSize is how many work blocks happen before a long break (the config's
	// long_break_every). It is 0 in a manual run, where there is no "set".
	SetSize int

	// SetPosition is, for a Work step, its 1-based position within the current
	// set (1..SetSize); for a break it is the position of the work block that
	// just finished. It is 0 when SetSize is 0.
	SetPosition int
}

// Provider yields the steps of a run in order. A manual run returns a single
// step and is then exhausted; an automatic run never runs out.
type Provider interface {
	// Next returns the next step, or ok == false when the run is complete.
	Next() (step Step, ok bool)
}

// Manual returns a one-shot provider: a single step of the given kind and
// duration, after which the run is complete.
func Manual(k Kind, d time.Duration) Provider {
	return &manual{step: Step{Kind: k, Duration: d, Ordinal: 1}}
}

type manual struct {
	step Step
	done bool
}

func (m *manual) Next() (Step, bool) {
	if m.done {
		return Step{}, false
	}
	m.done = true
	return m.step, true
}

// Auto returns a provider that walks the classic Pomodoro cycle forever:
// work, short break, work, short break, ... with a long break substituted in
// after every cfg.LongBreakEvery-th work block.
func Auto(cfg config.Config) Provider {
	return &auto{cfg: cfg}
}

type auto struct {
	cfg           config.Config
	workStarted   int  // number of work blocks handed out so far
	breaksStarted int  // number of breaks handed out so far
	expectBreak   bool // true once a work block has been handed out and not yet followed by its break
}

func (a *auto) Next() (Step, bool) {
	perSet := max(a.cfg.LongBreakEvery, 1)

	if !a.expectBreak {
		a.expectBreak = true
		a.workStarted++
		return Step{
			Kind:        Work,
			Duration:    a.cfg.WorkDuration(),
			Ordinal:     a.workStarted,
			SetSize:     perSet,
			SetPosition: positionInSet(a.workStarted, perSet),
		}, true
	}

	// The work block handed out last time is now finished; queue its break.
	a.expectBreak = false
	a.breaksStarted++
	pos := positionInSet(a.workStarted, perSet)

	step := Step{
		Duration:    a.cfg.ShortBreakDuration(),
		Kind:        ShortBreak,
		Ordinal:     a.breaksStarted,
		SetSize:     perSet,
		SetPosition: pos,
	}
	if pos == perSet { // just completed the last work block of the set
		step.Kind = LongBreak
		step.Duration = a.cfg.LongBreakDuration()
	}
	return step, true
}

// positionInSet maps a 1-based work-block index to its 1-based position
// within a set of size perSet (so the perSet-th block reports perSet, not 0).
func positionInSet(workIndex, perSet int) int {
	pos := workIndex % perSet
	if pos == 0 {
		return perSet
	}
	return pos
}
