package cfg

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Config Tags

var defaults = Tags{
	LokiURL:       "http://localhost:3100/loki/api/v1/push",
	LokiTimeout:   "3s",
	LokiRetries:   "5",
	LokiBatchWait: "200ms",
	LokiBatchSize: "250",

	TracingKind:         "jaeger",
	TracingURL:          "http://localhost:14268/api/traces",
	TracingLogsAsEvents: "true",
}

func (c Config) Get(key string) (string, error) {

	if opt, ok := os.LookupEnv("GT_" + key); ok {
		return opt, nil
	}
	if opt, ok := c[key]; ok {
		return opt, nil
	}
	if opt, ok := defaults[key]; ok {
		return opt, nil
	}
	return "", fmt.Errorf("no such option defined: %s", key)
}

func (c Config) Parse(filename string, r io.Reader) (Config, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return Config{}, fmt.Errorf("error reading config: %w", err)
	}

	lines := strings.Split(string(b), "\n")

	for i, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		err := Tags(c).Set(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s:%d: %w", filename, i+1, err)
		}
	}

	return c, nil
}
