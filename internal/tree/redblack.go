package tree

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/exp/constraints"
)

type RedBlack[K constraints.Ordered, V any] struct {
	lock sync.Mutex
	root *Node[K, V]
}

func (t *RedBlack[K, V]) Node(key K) *Node[K, V] {
	t.lock.Lock()
	defer t.lock.Unlock()

	return get(t.root, key)
}

func (t *RedBlack[K, V]) Value(key K) (V, bool) {
	n := t.Node(key)
	if n == nil {
		var v V
		return v, false
	}
	return n.Value, true
}

func (t *RedBlack[K, V]) MinMaxKeys() (K, K) {
	var left K
	var right K
	if t.root == nil {
		return left, right
	}

	node := t.root
	for node != nil {
		left = node.Key
		node = node.left
	}
	node = t.root
	for node != nil {
		right = node.Key
		node = node.right
	}

	return left, right
}

func (t *RedBlack[K, V]) Insert(key K, value V) {
	t.lock.Lock()
	defer t.lock.Unlock()

	neue := &Node[K, V]{
		Key:   key,
		Value: value,
		red:   true,
	}

	if t.root == nil {
		neue.red = false
		t.root = neue
		return
	}

	t.root = insert(t.root, neue)
	t.root.red = false
}

func (t *RedBlack[K, V]) String() string {
	return t.root.String()
}

func (t *RedBlack[K, V]) Walk(ctx context.Context, fn func(K, V)) error {
	return t.LimitedWalk(ctx, fn, walkAll[K]())
}

type WalkOption[K constraints.Ordered] func(current K) (include bool, walkRight bool)

func WalkRange[K constraints.Ordered](lower, upper K) WalkOption[K] {
	return func(current K) (bool, bool) {
		switch {
		case current > upper:
			return false, false
		case current < lower:
			return false, true
		default:
			return true, false
		}
	}
}

func WalkPrefix(prefix string) WalkOption[string] {
	return func(current string) (bool, bool) {
		switch {
		case strings.HasPrefix(current, prefix):
			return true, false
		case current < prefix:
			return false, true
		default:
			return false, false
		}
	}
}

func walkAll[K constraints.Ordered]() WalkOption[K] {
	return func(_ K) (bool, bool) {
		return true, false
	}
}

func (t *RedBlack[K, V]) LimitedWalk(ctx context.Context, fn func(K, V), option WalkOption[K]) error {
	if t.root == nil {
		return nil
	}

	stack := &Stack[*Node[K, V]]{}
	var current *Node[K, V]

	current = t.root
	for current != nil || stack.Len() != 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// No-op
		}

		if current == nil {
			val, ok := stack.Pop()
			if !ok {
				break
			}
			fn(val.Key, val.Value)
			current = val.right
		} else {
			push, right := option(current.Key)
			if push {
				stack.Push(current)
			}
			if right {
				current = current.right
			} else {
				current = current.left
			}
		}
	}

	return nil
}

type Node[K constraints.Ordered, V any] struct {
	Key   K
	Value V

	left  *Node[K, V]
	right *Node[K, V]
	red   bool
}

func (n *Node[K, V]) String() string {
	color := 'B'
	if n.red {
		color = 'R'
	}

	return fmt.Sprintf("Node[%c:%v](%v, %v)", color, n.Key, n.left, n.right)
}

func get[K constraints.Ordered, V any](parent *Node[K, V], key K) *Node[K, V] {
	if parent == nil {
		return nil
	}

	if key < parent.Key {
		return get(parent.left, key)
	}
	if key > parent.Key {
		return get(parent.right, key)
	}
	return parent
}

func insert[K constraints.Ordered, V any](parent, neue *Node[K, V]) *Node[K, V] {
	if neue.Key < parent.Key {
		if parent.left == nil {
			parent.left = neue
		} else {
			parent.left = insert(parent.left, neue)
		}
	}
	if neue.Key > parent.Key {
		if parent.right == nil {
			parent.right = neue
		} else {
			parent.right = insert(parent.right, neue)
		}
	}
	if neue.Key == parent.Key {
		parent.Value = neue.Value
	}

	if l, r := parent.Colors(); r && !l {
		parent = parent.RotLeft()
	}
	if l, _ := parent.Colors(); l {
		if l, _ = parent.left.Colors(); l {
			parent = parent.RotRight()
		}
	}
	if l, r := parent.Colors(); l && r {
		parent.ColorFlip()
	}

	return parent
}

func (n *Node[K, V]) Colors() (bool, bool) {
	return n.left != nil && n.left.red, n.right != nil && n.right.red
}

func (n *Node[K, V]) RotLeft() *Node[K, V] {
	child := n.right
	n.right = child.left
	child.left = n
	child.red = n.red
	n.red = true
	return child
}

func (n *Node[K, V]) RotRight() *Node[K, V] {
	child := n.left
	n.left = child.right
	child.right = n
	child.red = n.red
	n.red = true
	return child
}

func (n *Node[K, V]) ColorFlip() {
	n.red = true
	n.left.red = false
	n.right.red = false
}
