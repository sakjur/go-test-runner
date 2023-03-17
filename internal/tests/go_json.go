package tests

import (
	"bufio"
	"encoding/json"
	"io"
	"time"
)

type goTestLine struct {
	Time    time.Time
	Action  string
	Package string
	Elapsed float64
	Output  string
	Test    string
}

func (l goTestLine) Events() []Event {
	e := Event{
		Package:   l.Package,
		Test:      l.Test,
		Timestamp: l.Time,
	}

	switch l.Action {
	case "output":
		e.Payload = Print{Line: l.Output}
	case "start":
		e.Payload = StateChange{NewState: StateRunning}
	case "skip":
		e.Payload = StateChange{NewState: StateSkipped}
	case "pass":
		e.Payload = StateChange{NewState: StatePassed}
	case "fail":
		e.Payload = StateChange{NewState: StateFailed}
	}

	return []Event{e}
}

type GoJSON struct {
	scanner *bufio.Scanner
}

func NewGoJSON(r io.Reader) *GoJSON {
	return &GoJSON{scanner: bufio.NewScanner(r)}
}

func (j *GoJSON) ReadLine() ([]Event, error) {
	if !j.scanner.Scan() {
		return nil, io.EOF
	}
	line := goTestLine{}
	err := json.Unmarshal(j.scanner.Bytes(), &line)
	if err != nil {
		return nil, err
	}
	return line.Events(), nil
}
