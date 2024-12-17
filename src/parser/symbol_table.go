package parser

type Symtab[T any] struct {
	cur   int
	stack []map[string]T
}

func NewSymtab[T any]() *Symtab[T] {
	v := &Symtab[T]{cur: -1}
	v.Scope()
	return v
}

// Use to set current scope
func (v *Symtab[T]) Scope() {
	v.cur++
	v.stack = append(v.stack, make(map[string]T))
}

func (v *Symtab[T]) Unscope() {
	v.stack = v.stack[:v.cur]
	v.cur--
}

func (v *Symtab[T]) Set(k string, t T) {
	v.stack[v.cur][k] = t
}

func (v *Symtab[T]) Get(k string) (T, bool) {
	for i := v.cur; i >= 0; i-- {
		val, ok := v.stack[i][k]
		if ok {
			return val, true
		}
	}

	return *new(T), false
}
