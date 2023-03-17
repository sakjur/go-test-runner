package cfg

import (
	"errors"
	"fmt"
)

const (
	ConsoleLevel = "CONSOLE_LEVEL"
)

type PrintLevel int

const (
	PrintLevelUnknown PrintLevel = iota
	PrintLevelRaw
	PrintLevelNone
)

func (l PrintLevel) String() string {
	switch l {
	case PrintLevelRaw:
		return "raw"
	case PrintLevelNone:
		return "none"
	default:
		return "unknown"
	}
}

func printLevelFrom(s string) PrintLevel {
	switch s {
	case "raw":
		return PrintLevelRaw
	case "none":
		return PrintLevelNone
	default:
		return PrintLevelUnknown
	}
}

type ConsoleOptions struct {
	PrintLevel PrintLevel
}

func (c Config) Console() (ConsoleOptions, error) {
	rawLevel, levelErr := c.Get(ConsoleLevel)

	if err := errors.Join(levelErr); err != nil {
		return ConsoleOptions{}, fmt.Errorf("failed to get console configuration options: %w", err)
	}

	level := printLevelFrom(rawLevel)
	if level == PrintLevelUnknown {
		levelErr = fmt.Errorf("unknown console print level '%s', expected (raw|none)", rawLevel)
	}

	if err := errors.Join(levelErr); err != nil {
		return ConsoleOptions{}, fmt.Errorf("failed to parse console configuration options: %w", err)
	}

	return ConsoleOptions{PrintLevel: level}, nil
}
