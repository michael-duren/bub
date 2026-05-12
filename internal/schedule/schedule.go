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
	return &manual{step: Step{Kind: k, Duration: d}}
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
	workCompleted int  // number of work blocks finished so far
	expectBreak   bool // true once a work block has been handed out
}

func (a *auto) Next() (Step, bool) {
	if !a.expectBreak {
		a.expectBreak = true
		return Step{Kind: Work, Duration: a.cfg.WorkDuration()}, true
	}

	// We just handed out a work block last time; that block is now done.
	a.workCompleted++
	a.expectBreak = false

	if a.cfg.LongBreakEvery > 0 && a.workCompleted%a.cfg.LongBreakEvery == 0 {
		return Step{Kind: LongBreak, Duration: a.cfg.LongBreakDuration()}, true
	}
	return Step{Kind: ShortBreak, Duration: a.cfg.ShortBreakDuration()}, true
}
