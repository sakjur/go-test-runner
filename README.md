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
LOKI_URL="http://localhost:3100/loki/api/v1/push"
LOKI_TIMEOUT="3s"
LOKI_RETRIES="5",
LOKI_BATCH_WAIT="200ms"
LOKI_BATCH_SIZE="250"

TRACING_KIND="jaeger"
TRACING_URL="http://localhost:14268/api/traces"
TRACING_LOGS_AS_EVENTS="true"
```

All options can also be overridden using environment variables by
prefixing the key with `GT_` for the environmental variable.