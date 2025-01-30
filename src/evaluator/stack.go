package evaluator

import (
	"dash-lang.io/src/internal"
	"dash-lang.io/src/types"
)

type Stack struct {
	typs *internal.Cache[string, types.TypeSpec]
	vars *internal.StackedSymTab[any]
	prev *Stack
}

func NewStack(prev *Stack) *Stack {
	return &Stack{
		vars: internal.NewStackedSymbolTable[any](),
		prev: prev,
		typs: internal.NewCache[string, types.TypeSpec](),
	}
}

func (s *Stack) Set(v string, k any) {
	s.vars.Set(v, k)
}

func (s *Stack) Get(v string) (any, bool) {
	val, ok := s.vars.Get(v)
	if !ok {
		if s.prev == nil {
			return nil, false
		}
		return s.prev.Get(v)
	}
	return val, true
}

// Creates a new scope within the stack to prevent
// variables from overlapping within different scopes
// in program e.g. vars defined in if else block
func (s *Stack) Scope() {
	s.vars.Scope()
}

func (s *Stack) Unscope() {
	s.vars.Unscope()
}
