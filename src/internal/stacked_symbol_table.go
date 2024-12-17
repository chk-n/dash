package internal

type StackedSymTab[T any] struct {
	// current scope
	cur int64
	stk []map[string]T
}

func NewStackedSymbolTable[T any]() *StackedSymTab[T] {
	s := &StackedSymTab[T]{
		cur: -1,
	}

	s.Scope()

	return s
}

// Use to set current scope
func (s *StackedSymTab[T]) Scope() {
	s.cur++
	s.stk = append(s.stk, make(map[string]T))
}

func (s *StackedSymTab[T]) Unscope() {
	s.stk = s.stk[:s.cur]
	s.cur--
}

func (s *StackedSymTab[T]) SetIn(scope int64, vr string, v T) {
	s.stk[scope][vr] = v
}

func (s *StackedSymTab[T]) Set(vr string, v T) {
	s.stk[s.cur][vr] = v
}

func (s *StackedSymTab[T]) Get(vr string) (T, bool) {
	for i := s.cur; i >= 0; i-- {
		val, ok := s.stk[i][vr]
		if ok {
			return val, true
		}
	}

	return *new(T), false
}

func (s *StackedSymTab[T]) Clear(vr string) {
	for i := s.cur; i >= 0; i-- {
		if _, ok := s.stk[i][vr]; ok {
			delete(s.stk[i], vr)
			return
		}
	}
}

// ------------- //
// For debugging //
// ------------- //

func (s *StackedSymTab[T]) GetAll() []map[string]T {
	return s.stk
}

func (s *StackedSymTab[T]) GetI() int64 {
	return s.cur
}
