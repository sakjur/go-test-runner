# go test runner

_beware, this is a hackyhackhack, and not ready for general purpose use_

Send JSON formatted Go test results to Tempo and Loki to enable analysis
of test results using Grafana+Loki+Tempo.

### Quickstart
```bash
docker compose -f ./dev/docker-compose.yaml up -d
go build .
export GOTR=$(pwd)

cd YOUR_GO_PROJECT
go test -json ./... | $GOTR/go-test-runner -t PR=123 -t Author=emil@example.org
```

### Configuration

```bash
go-test-runner -c configuration-file
```

`configuration-file` is a = new-line delimited separated
list of `keys=values` for the various configuration options.

Default values:
```
## Options for sending logs to Loki
# URL for Loki's ingestor
LOKI_URL="http://localhost:3100/loki/api/v1/push"
# Timeout for requests to Loki
LOKI_TIMEOUT="3s"
# Number of retries before giving up on sending logs to Loki
LOKI_RETRIES="5"
# Wait between sending batches
LOKI_BATCH_WAIT="200ms"
# Max number of events per batch
LOKI_BATCH_SIZE="250"

## Options for sending traces to Tempo or other distributed tracing system
# Protocol with which to send traces
# - jaeger: Use the Jaeger protocol
TRACING_KIND="jaeger"
# URL for the distributed tracing system
TRACING_URL="http://localhost:14268/api/traces"
# Option for adding testing logs to the spans
# - true: All logs are sent as events, this option may lead to very large traces
# - false: No logs are sent as events
TRACING_LOGS_AS_EVENTS="false"

## Options for printing to standard output
# How much should be printed to the console?
# - raw: same output as a regular "go test" run
# - none: don't print anything per test, only the final summary
CONSOLE_LEVEL="raw"

## Options for connecting to Grafana
# URL to the index of the Grafana instance from where Loki logs can be retrieved. 
GRAFANA_URL="http://localhost:3000/"
# The name of the Loki data source to use for Explore.
GRAFANA_LOKI_DATASOURCE="loki"
# The UID of the Loki data source to use for Explore.
GRAFANA_LOKI_DATASOURCE_UID="loki"
```

All options can also be overridden using environment variables by
prefixing the key with `GT_` for the environmental variable.