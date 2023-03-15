package cfg

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

const (
	LokiURL       = "LOKI_URL"
	LokiTimeout   = "LOKI_TIMEOUT"
	LokiRetries   = "LOKI_RETRIES"
	LokiBatchWait = "LOKI_BATCH_WAIT"
	LokiBatchSize = "LOKI_BATCH_SIZE"
)

type LokiOptions struct {
	URL       string
	Timeout   time.Duration
	Retries   int
	BatchWait time.Duration
	BatchSize int
}

func (c Config) Loki() (LokiOptions, error) {
	url, urlErr := c.Get(LokiURL)
	rawTimeout, rawTimeoutErr := c.Get(LokiTimeout)
	rawRetries, rawRetriesErr := c.Get(LokiRetries)
	rawBatchWait, rawBatchWaitErr := c.Get(LokiBatchWait)
	rawBatchSize, rawBatchSizeErr := c.Get(LokiBatchSize)

	if err := errors.Join(urlErr, rawTimeoutErr, rawRetriesErr, rawBatchSizeErr, rawBatchWaitErr); err != nil {
		return LokiOptions{}, fmt.Errorf("failed to get Loki configuration options: %w", err)
	}

	timeout, timeoutErr := time.ParseDuration(rawTimeout)
	batchWait, batchWaitErr := time.ParseDuration(rawBatchWait)
	retries, retriesErr := strconv.Atoi(rawRetries)
	batchSize, batchSizeErr := strconv.Atoi(rawBatchSize)

	if err := errors.Join(timeoutErr, retriesErr, batchWaitErr, batchSizeErr); err != nil {
		return LokiOptions{}, fmt.Errorf("failed to parse Loki configuration options: %w", err)
	}

	return LokiOptions{
		URL:       url,
		Timeout:   timeout,
		Retries:   retries,
		BatchSize: batchSize,
		BatchWait: batchWait,
	}, nil
}
