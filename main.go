// Command bub is a small Pomodoro timer for the terminal, built with
// Bubble Tea and the Bubbles progress component.
//
// Manual mode runs a single block:
//
//	bub work 25
//	bub break 5
//	bub 50            // shorthand for "bub work 50"
//
// Automatic mode loops work -> break on the classic Pomodoro schedule and is
// configurable via ~/.config/.bub.yaml:
//
//	bub
//	bub auto
package main

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/michael-duren/bub/internal/config"
	"github.com/michael-duren/bub/internal/schedule"
	"github.com/michael-duren/bub/internal/tui"
)

//go:embed img/bub-alert.png
var alertIcon []byte

const usage = `bub — a Pomodoro timer

USAGE
  bub                    Automatic mode: loop work -> break on the Pomodoro
                         schedule. Reads ~/.config/.bub.yaml if present.
  bub auto               Same as "bub" with no arguments.
  bub work [minutes]     Run one work block (default: 25 minutes).
  bub break [minutes]    Run one break (default: 5 minutes). "rest" works too.
  bub <minutes>          Shorthand for "bub work <minutes>".
  bub -h | --help        Show this help.

KEYS (while a timer is running)
  space / p     pause or resume
  s             skip to the next step
  r             restart the current step
  q / ctrl+c    quit

CONFIG  (~/.config/.bub.yaml — every field optional, falls back to defaults)
  work_minutes: 25
  short_break_minutes: 5
  long_break_minutes: 15
  long_break_every: 4        # take a long break after every Nth work block
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "bub: "+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help", "help":
			fmt.Print(usage)
			return nil
		}
	}

	provider, err := buildProvider(args)
	if err != nil {
		return err
	}

	m, ok := tui.New(provider, alertIcon)
	if !ok {
		return nil // nothing scheduled
	}

	_, err = tea.NewProgram(m).Run()
	return err
}

func buildProvider(args []string) (schedule.Provider, error) {
	// No args (or "auto") => automatic Pomodoro loop driven by config.
	if len(args) == 0 || args[0] == "auto" {
		cfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
		return schedule.Auto(cfg), nil
	}

	switch args[0] {
	case "work":
		return manualProvider(schedule.Work, 25, args[1:])
	case "break", "rest":
		return manualProvider(schedule.ShortBreak, 5, args[1:])
	}

	// "bub 25" => single 25-minute work block.
	if minutes, err := strconv.Atoi(args[0]); err == nil {
		if minutes <= 0 {
			return nil, fmt.Errorf("duration must be a positive number of minutes, got %d", minutes)
		}
		return schedule.Manual(schedule.Work, time.Duration(minutes)*time.Minute), nil
	}

	return nil, fmt.Errorf("unknown command %q (run \"bub --help\")", args[0])
}

func manualProvider(kind schedule.Kind, defaultMinutes int, rest []string) (schedule.Provider, error) {
	minutes := defaultMinutes
	if len(rest) > 0 {
		v, err := strconv.Atoi(rest[0])
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("invalid duration %q: want a positive number of minutes", rest[0])
		}
		minutes = v
	}
	return schedule.Manual(kind, time.Duration(minutes)*time.Minute), nil
}
