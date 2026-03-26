package evaluator

import (
	"dash-lang.io/src/ast"
	"dash-lang.io/src/internal"
	"dash-lang.io/src/types"
)

type Context struct {
	// Keeps track of libraries imported by current library
	imps *internal.Cache[string, *ast.Library]
	// Keeps track of all type defs, aliases, unions, and structs
	// available in current context
	typs *internal.Cache[string, types.Type]
	// stores variables, function closures, functions
	vars *internal.StackedSymTab[any]
	prev *Context
	// Current library path (e.g., "dash/lexer").
	// NOTE: its always set on every context as it
	// is needed to generate type descriptor e.g.
	// for match or any types
	libPath string
}

func NewContext(prev *Context) *Context {
	if prev == nil {
		return &Context{
			imps: internal.NewCache[string, *ast.Library](),
			vars: internal.NewStackedSymbolTable[any](),
			typs: internal.NewCache[string, types.Type](),
		}
	}
	return &Context{
		imps:    prev.imps,
		typs:    prev.typs,
		vars:    internal.NewStackedSymbolTable[any](),
		prev:    prev,
		libPath: prev.libPath,
	}
}

// NewContextWith creates a new context with an explicit library path
func NewContextWith(prev *Context, libPath string) *Context {
	return &Context{
		imps:    internal.NewCache[string, *ast.Library](),
		vars:    internal.NewStackedSymbolTable[any](),
		prev:    prev,
		typs:    internal.NewCache[string, types.Type](),
		libPath: libPath,
	}
}

// Set key to value in all stacked symbol tables in current context
func (c *Context) SetAll(v string, k any) {
	for i := c.vars.GetI(); i >= 0; i-- {
		c.vars.SetIn(i, v, k)
	}
	// note: we dont propagate the previous context
	// as that is outside current boundary e.g. global
	// or another function encapsulating current context
}

func (c *Context) Set(v string, k any) {
	c.vars.Set(v, k)
}

func (c *Context) Get(v string) (any, bool) {
	val, ok := c.vars.Get(v)
	if !ok {
		if c.prev == nil {
			return nil, false
		}
		return c.prev.Get(v)
	}
	return val, true
}

// GetType looks up a type in the current context and parent contexts
func (c *Context) GetType(name string) (types.Type, bool) {
	typ, ok := c.typs.Get(name)
	if !ok {
		if c.prev == nil {
			return nil, false
		}
		return c.prev.GetType(name)
	}
	return typ, true
}

// Creates a new scope within the contexrt to prevent
// variables from overlapping within different scopes
// in program e.g. vars defined in if else block
func (c *Context) Scope() {
	c.vars.Scope()
}

func (c *Context) Unscope() {
	c.vars.Unscope()
}

func (c *Context) Release() {
	// no-op: contexts may be captured by closures that outlive
	// the defining scope, so pooling is unsafe.
}
