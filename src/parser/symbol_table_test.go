package parser

import (
	"testing"

	"dash-lang.io/src/token"
)

func TestNewSymtab(t *testing.T) {
	symtab := NewSymtab[token.Type]()
	if len(symtab.stack) != 1 {
		t.Errorf("Expected stack length of 1, got %d", len(symtab.stack))
	}
}

func TestScopeAndUnscope(t *testing.T) {
	symtab := NewSymtab[token.Type]()
	symtab.Scope() // Enter a new scope
	if len(symtab.stack) != 2 {
		t.Errorf("Expected stack length of 2 after entering a scope, got %d", len(symtab.stack))
	}

	symtab.Unscope() // Exit the scope
	if len(symtab.stack) != 1 {
		t.Errorf("Expected stack length of 1 after exiting a scope, got %d", len(symtab.stack))
	}
}

func TestSetAndGet(t *testing.T) {
	symtab := NewSymtab[token.Type]()
	symtab.Set("x", token.INT)
	val, exists := symtab.Get("x")
	if !exists || val != token.INT {
		t.Errorf("Expected to get token.INT for 'x', got %v, exists: %t", val, exists)
	}
}

func TestShadowing(t *testing.T) {
	symtab := NewSymtab[token.Type]()
	symtab.Set("x", token.INT)    // Set in global scope
	symtab.Scope()                // Enter a new scope
	symtab.Set("x", token.STRING) // Shadow 'x' in the new scope

	val, _ := symtab.Get("x")
	if val != token.STRING {
		t.Errorf("Expected to get token.STRING for 'x' in inner scope, got %v", val)
	}

	symtab.Unscope() // Exit to global scope
	val, _ = symtab.Get("x")
	if val != token.INT {
		t.Errorf("Expected to get token.INT for 'x' in global scope, got %v", val)
	}
}

func TestGetNonExistent(t *testing.T) {
	symtab := NewSymtab[token.Type]()
	_, exists := symtab.Get("nonexistent")
	if exists {
		t.Error("Expected exists to be false for a nonexistent symbol")
	}
}
