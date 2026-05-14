// Package tui is the Bubble Tea program: it counts down the current step and
// renders it with the Bubbles progress component, then asks the schedule
// provider for the next step.
package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/michael-duren/bub/internal/schedule"
)

const tickInterval = time.Second

// progress bar fill colours
const (
	workFill  = "#7D56F4" // purple
	breakFill = "#04B575" // green
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	timeStyle  = lipgloss.NewStyle().Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// ringBell emits a terminal bell so a step change is noticeable when the
// window isn't focused.
func ringBell() tea.Cmd {
	return func() tea.Msg {
		fmt.Fprint(os.Stderr, "\a")
		return nil
	}
}

// sendNotification sends a macOS push notification when running on darwin.
// It blocks until the notifier exits, so it is safe to call before tea.Quit.
// If terminal-notifier is available and iconBytes is non-nil, the custom icon
// is written to a temp file and passed as -contentImage.
func sendNotification(title, body string, iconBytes []byte) {
	if runtime.GOOS != "darwin" {
		return
	}

	// Prefer terminal-notifier for custom icon support.
	if path, err := exec.LookPath("terminal-notifier"); err == nil {
		args := []string{"-title", title, "-message", body, "-sender", "com.apple.Terminal", "-sound", "default"}
		if len(iconBytes) > 0 {
			tmp, err := os.CreateTemp("", "bub-icon-*.png")
			if err == nil {
				if _, werr := tmp.Write(iconBytes); werr == nil {
					tmp.Close()
					args = append(args, "-contentImage", tmp.Name())
				} else {
					tmp.Close()
					os.Remove(tmp.Name())
				}
			}
		}
		//nolint:errcheck
		exec.Command(path, args...).Run()
		return
	}

	// Fallback: plain osascript (no custom icon).
	script := fmt.Sprintf(`display notification %q with title %q`, body, title)
	//nolint:errcheck
	exec.Command("osascript", "-e", script).Run()
}

// notify wraps sendNotification as a tea.Cmd for use in the normal (non-final)
// step transition, where Bubble Tea will schedule it as usual.
func notify(title, body string, iconBytes []byte) tea.Cmd {
	return func() tea.Msg {
		sendNotification(title, body, iconBytes)
		return nil
	}
}

// Model is the Bubble Tea model for a bub run.
type Model struct {
	provider  schedule.Provider
	step      schedule.Step
	iconBytes []byte // embedded alert icon for macOS notifications

	progress progress.Model
	elapsed  time.Duration
	paused   bool

	width    int
	height   int
	workDone int  // number of work blocks completed this run
	finished bool // run is over (manual one-shot done)
}

// New builds a Model from a schedule provider. It returns ok == false when the
// provider has no steps at all. iconBytes is the PNG to show in macOS
// notifications; pass nil to use the system default.
func New(p schedule.Provider, iconBytes []byte) (Model, bool) {
	step, ok := p.Next()
	if !ok {
		return Model{}, false
	}
	return Model{
		provider:  p,
		step:      step,
		iconBytes: iconBytes,
		progress:  newBar(step.Kind, 0),
	}, true
}

func newBar(k schedule.Kind, width int) progress.Model {
	fill := workFill
	if k != schedule.Work {
		fill = breakFill
	}
	bar := progress.New(progress.WithSolidFill(fill))
	if width > 0 {
		bar.Width = width
	}
	return bar
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, tick())
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case " ", "p":
			m.paused = !m.paused
			return m, nil
		case "s": // skip to the next step
			m, cmd := m.advance()
			return m, cmd
		case "r": // restart the current step
			m.elapsed = 0
			return m, m.progress.SetPercent(0)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		w := max(min(msg.Width/2, 80), 12)
		m.progress.Width = w
		return m, nil

	case tickMsg:
		if m.finished {
			return m, nil
		}
		if m.paused {
			return m, tick()
		}

		m.elapsed += tickInterval
		if m.elapsed >= m.step.Duration {
			m.elapsed = m.step.Duration
			done := m.step
			if done.Kind == schedule.Work {
				m.workDone++
			}
			nextModel, advCmd := m.advance()
			notifTitle := headline(done) + " done"
			notifBody := clock(done.Duration) + " elapsed"
			if !nextModel.finished {
				notifBody += " · up next: " + label(nextModel.step.Kind)
			}
			if nextModel.finished {
				// Run is over — send the notification synchronously before
				// tea.Quit is processed, otherwise Bubble Tea exits too fast.
				sendNotification(notifTitle, notifBody, m.iconBytes)
			}
			return nextModel, tea.Batch(
				tea.Printf("✓  %s  ·  %s", headline(done), clock(done.Duration)),
				ringBell(),
				notify(notifTitle, notifBody, m.iconBytes),
				advCmd,
				tick(),
			)
		}

		pct := float64(m.elapsed) / float64(m.step.Duration)
		return m, tea.Batch(tick(), m.progress.SetPercent(pct))

	case progress.FrameMsg:
		updated, cmd := m.progress.Update(msg)
		m.progress = updated.(progress.Model)
		return m, cmd
	}

	return m, nil
}

// advance moves to the next step from the provider. If the provider is
// exhausted (manual run) the program quits.
func (m Model) advance() (Model, tea.Cmd) {
	next, ok := m.provider.Next()
	if !ok {
		m.finished = true
		return m, tea.Quit
	}
	width := m.progress.Width
	m.step = next
	m.elapsed = 0
	m.progress = newBar(next.Kind, width)
	return m, m.progress.SetPercent(0)
}

// View implements tea.Model.
func (m Model) View() string {
	remaining := max(m.step.Duration-m.elapsed, 0)

	var b strings.Builder
	b.WriteString("\n  ")
	b.WriteString(titleStyle.Render(headline(m.step)))
	if m.paused {
		b.WriteString(dimStyle.Render("   ⏸ paused"))
	}
	if d := detail(m.step); d != "" {
		b.WriteString("\n  ")
		b.WriteString(dimStyle.Render(d))
	}

	b.WriteString("\n\n  ")
	b.WriteString(m.progress.View())

	b.WriteString("\n\n  ")
	b.WriteString(timeStyle.Render(clock(remaining)))
	b.WriteString(dimStyle.Render("  /  " + clock(m.step.Duration)))
	if m.workDone > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("   ·   %d pomodoro%s done", m.workDone, plural(m.workDone))))
	}

	b.WriteString("\n\n  ")
	b.WriteString(dimStyle.Render("space pause · s skip · r restart · q quit"))
	b.WriteString("\n")

	if m.width == 0 || m.height == 0 {
		return b.String()
	}
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		b.String(),
	)
}

// headline is the "🍅  Focus" line for a step. In automatic mode (SetSize > 0)
// it carries the running session number, e.g. "🍅  Focus #3" / "☕  Break #2".
func headline(s schedule.Step) string {
	if s.SetSize > 0 {
		return fmt.Sprintf("%s  %s #%d", icon(s.Kind), label(s.Kind), s.Ordinal)
	}
	return fmt.Sprintf("%s  %s", icon(s.Kind), label(s.Kind))
}

// detail says where the step sits in the Pomodoro cycle, e.g.
// "3 of 4 before a long break". It is empty in manual mode.
func detail(s schedule.Step) string {
	if s.SetSize <= 1 {
		return ""
	}
	switch s.Kind {
	case schedule.Work:
		return fmt.Sprintf("%d of %d before a long break", s.SetPosition, s.SetSize)
	case schedule.LongBreak:
		return "set complete — fresh start next"
	default: // short break
		left := s.SetSize - s.SetPosition
		return fmt.Sprintf("%d pomodoro%s until a long break", left, plural(left))
	}
}

func label(k schedule.Kind) string {
	switch k {
	case schedule.Work:
		return "Focus"
	case schedule.LongBreak:
		return "Long break"
	default:
		return "Break"
	}
}

func icon(k schedule.Kind) string {
	switch k {
	case schedule.Work:
		return "🍅"
	case schedule.LongBreak:
		return "🌴"
	default:
		return "☕"
	}
}

func clock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Round(time.Second).Seconds())
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
