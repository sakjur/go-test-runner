package cfg

import (
	"errors"
	"fmt"
)

const (
	GrafanaURL               = "GRAFANA_URL"
	GrafanaLokiDatasource    = "GRAFANA_LOKI_DATASOURCE"
	GrafanaLokiDatasourceUID = "GRAFANA_LOKI_DATASOURCE_UID"
)

type GrafanaOptions struct {
	URL               string
	LokiDatasource    string
	LokiDatasourceUID string
}

func (c Config) Grafana() (GrafanaOptions, error) {
	url, urlErr := c.Get(GrafanaURL)
	lokiDS, lokiDSErr := c.Get(GrafanaLokiDatasource)
	lokiUID, lokiUIDErr := c.Get(GrafanaLokiDatasourceUID)

	if err := errors.Join(urlErr, lokiDSErr, lokiUIDErr); err != nil {
		return GrafanaOptions{}, fmt.Errorf("failed to get Grafana configuration options: %w", err)
	}

	return GrafanaOptions{
		URL:               url,
		LokiDatasource:    lokiDS,
		LokiDatasourceUID: lokiUID,
	}, nil
}
