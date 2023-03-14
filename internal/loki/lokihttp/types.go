package lokihttp

import (
	"github.com/prometheus/common/model"

	"github.com/grafana/go-test-runner/internal/loki/logproto"
)

// Entry is a log entry with labels.
type Entry struct {
	Labels model.LabelSet
	logproto.Entry
}
