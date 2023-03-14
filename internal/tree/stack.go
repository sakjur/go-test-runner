package tree

import "sync"

type Stack[T any] struct {
	lock  sync.Mutex
	stack []T
}

func (s *Stack[T]) Len() int {
	return len(s.stack)
}

func (s *Stack[T]) Push(item T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.stack = append(s.stack, item)
}

func (s *Stack[T]) Pop() (T, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var item T
	count := len(s.stack)
	if count == 0 {
		return item, false
	}

	item = s.stack[count-1]
	s.stack = s.stack[:count-1]
	return item, true
}

func (s *Stack[T]) Peek() (T, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var item T
	count := len(s.stack)
	if count == 0 {
		return item, false
	}

	item = s.stack[count-1]
	return item, true
}
