package cfg

import (
	"fmt"
	"strconv"
	"strings"
)

type Tags map[string]string

func (t Tags) String() string {
	ts := make([]string, 0, len(t))
	for key, value := range t {
		ts = append(ts, fmt.Sprintf("-t %s=%s", key, strconv.Quote(value)))
	}
	return strings.Join(ts, " ")
}

func (t Tags) Set(s string) error {
	values := strings.SplitN(s, "=", 2)
	if len(values) != 2 {
		return fmt.Errorf("expected tags to have the format 'key=value'")
	}

	key := values[0]
	val := values[1]

	if strings.Contains(key, " ") {
		return fmt.Errorf("keys must not contain spaces")
	}

	if strings.HasPrefix(val, "\"") {
		var err error
		val, err = strconv.Unquote(val)
		if err != nil {
			return fmt.Errorf("failed to unquote value for tag %s", key)
		}
	}
	t[key] = val
	return nil
}
