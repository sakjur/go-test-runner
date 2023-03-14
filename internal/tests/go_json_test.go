package tests

import (
	"context"
	"go-test-runner/internal/tree"
	"io"
	"os"
	"testing"

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

	fn := func(_ string, c *Collection) {
		c.Tests.Walk(ctx, testWalk(t))
	}
	//r.Collection.Walk(context.Background(), fn)

	r.Collection.LimitedWalk(ctx, fn, tree.WalkPrefix("github.com/grafana/grafana/pkg/util"))
}

func testWalk(t *testing.T) func(k string, tst *Test) {
	t.Helper()
	return func(k string, tst *Test) {
		t.Log(tst.Package, tst.Name, tst.State.String())
		tst.Subtests.Walk(context.Background(), testWalk(t))
	}
}
