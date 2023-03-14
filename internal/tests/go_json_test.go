package tests

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/go-test-runner/internal/tree"

	"github.com/stretchr/testify/require"
)

func TestGfTestSuite(t *testing.T) {
	r := &Run{
		Collection: &tree.RedBlack[string, *Collection]{},
	}

	f, err := os.Open("testdata/grafana-test-suite.json")
	require.NoError(t, err)

	goJSON := NewGoJSON(f)
	for {
		events, err := goJSON.ReadLine()
		if err != nil {
			require.ErrorIs(t, err, io.EOF)
			break
		}
		r.Add(context.Background(), events...)
	}

	ctx := context.Background()

	prefix := "github.com/grafana/grafana/pkg/util"
	fn := func(_ string, c *Collection) {
		c.Tests.Walk(ctx, testWalk(t, prefix))
	}

	r.Collection.LimitedWalk(ctx, fn, tree.WalkPrefix(prefix))
}

func testWalk(t *testing.T, prefix string) func(k string, tst *Test) {
	t.Helper()
	return func(k string, tst *Test) {
		assert.True(t, strings.HasPrefix(tst.Package, prefix))
	}
}
