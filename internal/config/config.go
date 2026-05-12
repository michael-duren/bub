// Package config loads bub's optional automatic-schedule configuration from
// ~/.config/.bub.yaml. A missing file is fine — defaults are used.
package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config controls the automatic Pomodoro schedule.
type Config struct {
	// WorkMinutes is the length of a focused work block.
	WorkMinutes int `yaml:"work_minutes"`
	// ShortBreakMinutes is the break taken after most work blocks.
	ShortBreakMinutes int `yaml:"short_break_minutes"`
	// LongBreakMinutes is the longer break taken after a full set of blocks.
	LongBreakMinutes int `yaml:"long_break_minutes"`
	// LongBreakEvery is how many work blocks happen before a long break.
	LongBreakEvery int `yaml:"long_break_every"`
}

// Default returns the classic Pomodoro schedule: 25 min work, 5 min short
// break, and a 15 min long break after every 4th work block.
func Default() Config {
	return Config{
		WorkMinutes:       25,
		ShortBreakMinutes: 5,
		LongBreakMinutes:  15,
		LongBreakEvery:    4,
	}
}

// Path is where bub looks for its config file (~/.config/.bub.yaml on Linux).
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".bub.yaml"), nil
}

// Load reads the config file, filling any missing or non-positive field from
// Default. A missing file is not an error.
func Load() (Config, error) {
	cfg := Default()

	path, err := Path()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	var fromFile Config
	if err := yaml.Unmarshal(data, &fromFile); err != nil {
		return cfg, err
	}

	if fromFile.WorkMinutes > 0 {
		cfg.WorkMinutes = fromFile.WorkMinutes
	}
	if fromFile.ShortBreakMinutes > 0 {
		cfg.ShortBreakMinutes = fromFile.ShortBreakMinutes
	}
	if fromFile.LongBreakMinutes > 0 {
		cfg.LongBreakMinutes = fromFile.LongBreakMinutes
	}
	if fromFile.LongBreakEvery > 0 {
		cfg.LongBreakEvery = fromFile.LongBreakEvery
	}

	return cfg, nil
}

// WorkDuration is WorkMinutes as a time.Duration.
func (c Config) WorkDuration() time.Duration {
	return time.Duration(c.WorkMinutes) * time.Minute
}

// ShortBreakDuration is ShortBreakMinutes as a time.Duration.
func (c Config) ShortBreakDuration() time.Duration {
	return time.Duration(c.ShortBreakMinutes) * time.Minute
}

// LongBreakDuration is LongBreakMinutes as a time.Duration.
func (c Config) LongBreakDuration() time.Duration {
	return time.Duration(c.LongBreakMinutes) * time.Minute
}
