package explore

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type ExploreLink struct {
	GrafanaURL    string
	DataSource    string
	DataSourceUID string
	TraceID       string
}

func (x ExploreLink) URL() (*url.URL, error) {
	encoded, err := json.Marshal(query{
		DataSource: x.DataSource,
		Queries: []queryList{
			{
				RefID: "A",
				DataSource: datasource{
					Type: "loki",
					UID:  x.DataSourceUID,
				},
				EditorMode: "code",
				QueryType:  "range",
				Expr:       fmt.Sprintf("{source=\"go-test-runner\"} | logfmt | traceID=\"%s\" | line_format \"{{ .msg }}\"", x.TraceID),
			},
		},
		Range: timeRange{
			From: "now-1h",
			To:   "now",
		},
	})
	if err != nil {
		return nil, err
	}

	parsedURL, err := url.Parse(x.GrafanaURL)
	if err != nil {
		return nil, err
	}

	parsedURL.Path = "/explore"
	parsedURL.RawQuery = url.Values{"left": []string{string(encoded)}}.Encode()
	return parsedURL, nil
}

func (x ExploreLink) String() string {
	parsedURL, err := x.URL()
	if err != nil {
		panic(err)
	}
	return parsedURL.String()
}

type query struct {
	DataSource string      `json:"datasource"`
	Queries    []queryList `json:"queries"`
	Range      timeRange   `json:"range"`
}

type timeRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type queryList struct {
	RefID      string     `json:"refId"`
	DataSource datasource `json:"datasource"`
	EditorMode string     `json:"editorMode,omitempty"`
	QueryType  string     `json:"queryType"`
	Expr       string     `json:"expr"`
}

type datasource struct {
	Type string `json:"type"`
	UID  string `json:"uid"`
}
