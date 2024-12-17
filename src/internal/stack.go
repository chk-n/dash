package internal

type Stack[T comparable] struct {
	d []T
}

func NewStack[T comparable]() *Stack[T] {
	return &Stack[T]{}
}

func (q *Stack[T]) Len() int {
	return len(q.d)
}

func (q *Stack[T]) Push(v T) {
	q.d = append(q.d, v)
}

func (q *Stack[T]) Pop() T {
	v := q.d[len(q.d)-1]
	q.d = q.d[:len(q.d)-1]

	return v
}

func (q *Stack[T]) PopAll() []T {
	v := q.d
	q.d = nil

	return v
}

func (q *Stack[T]) GetAll() []T {
	return q.d
}

func (q *Stack[T]) GetLast() T {
	if len(q.d) == 0 {
		panic("no scope")
	}
	return q.d[len(q.d)-1]
}

func (q *Stack[T]) IsIn(a T) bool {
	for _, b := range q.d {
		if a == b {
			return true
		}
	}
	return false
}
