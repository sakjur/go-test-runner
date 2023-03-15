package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/grafana/go-test-runner/internal/cfg"

	"github.com/grafana/go-test-runner/internal/console"
	"github.com/grafana/go-test-runner/internal/loki"
	"github.com/grafana/go-test-runner/internal/tests"
)

type eventHandler interface {
	Handle(tests.Event) error
}

type stoppable interface {
	Stop()
}

func main() {
	conf := cfg.Config{}
	fields := cfg.Tags{}
	flag.Var(&fields, "t", "Add a key=value pair to the log output for each test")
	file := flag.String("c", "", "Path to configuration file")
	flag.Parse()

	if *file != "" {
		f, err := os.Open(*file)
		if err != nil {
			panic(err)
		}
		conf, err = conf.Parse(*file, f)
		if err != nil {
			panic(err)
		}
	}

	tracingOptions, err := conf.Tracing()
	if err != nil {
		panic(err)
	}

	r, err := tests.New(fields, tracingOptions)
	if err != nil {
		panic(err)
	}
	r.CollectionDivider = "/"

	lokiOptions, err := conf.Loki()
	if err != nil {
		panic(err)
	}

	logClient, err := loki.New(r, lokiOptions)
	if err != nil {
		panic(err)
	}

	handlers := []eventHandler{
		r,
		logClient,
		console.New(r.TraceID),
	}

	goJSON := tests.NewGoJSON(os.Stdin)
	for {
		es, err := goJSON.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "error parsing line from `go test -json`!")
			continue
		}
		for _, e := range es {
			for _, handler := range handlers {
				err := handler.Handle(e)
				if err != nil {
					fmt.Fprintf(os.Stderr, "got error from handler '%T': %v", handler, err)
				}
			}
		}
	}

	for _, handler := range handlers {
		if stopper, ok := handler.(stoppable); ok {
			stopper.Stop()
		}
	}
}
