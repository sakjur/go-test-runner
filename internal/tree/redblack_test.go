package tree

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/constraints"
	"testing"
)

func TestRB(t *testing.T) {
	tree := &RedBlack[string, struct{}]{}
	for _, c := range "abczyx" {
		tree.Insert(string([]rune{c}), struct{}{})
	}
	fmt.Printf("%v %v", tree, tree.blackBalance())
}

func FuzzRB_Balance(f *testing.F) {
	f.Add("")
	f.Add("abczyx")
	f.Add("hello world")

	f.Fuzz(func(t *testing.T, keys string) {
		tree := &RedBlack[rune, struct{}]{}
		for _, key := range keys {
			tree.Insert(key, struct{}{})
		}
		assert.True(t, tree.blackBalance())
	})
}

func (t *RedBlack[K, V]) blackBalance() bool {
	n := 0
	node := t.root
	for node != nil {
		if !node.red {
			n++
		}
		node = node.left
	}

	return blackBalance(t.root, n, 0)
}

func blackBalance[K constraints.Ordered, V any](parent *Node[K, V], expected, accum int) bool {
	if parent == nil {
		return expected == accum
	}
	if !parent.red {
		accum++
	}

	return blackBalance(parent.left, expected, accum) && blackBalance(parent.right, expected, accum)
}
