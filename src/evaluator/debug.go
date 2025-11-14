package evaluator

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// recoverPanic catches panics during evaluation and converts them to Error objects
// with stack trace information for debugging. Keep this function outside of
// evaluator.go so we can easily access line of panic by checking for "evaluator.go"
// in stack output
func recoverPanic(result *any) {
	if r := recover(); r != nil {
		var errMsg string
		switch v := r.(type) {
		case string:
			errMsg = v
		case error:
			errMsg = v.Error()
		default:
			errMsg = fmt.Sprintf("%v", r)
		}

		stack := string(debug.Stack())
		lines := strings.Split(stack, "\n")
		var location string
		for i := range lines {
			if strings.Contains(lines[i], "evaluator.go:") {
				location = strings.TrimSpace(lines[i])
				break
			}
		}

		if location != "" {
			*result = &Return{Values: []any{&Error{
				descriptor: generateTypeDescriptor("RuntimeError"),
				Err:        fmt.Sprintf("Runtime panic at %s: %s", location, errMsg),
			}}}
		} else {
			*result = &Return{Values: []any{&Error{
				descriptor: generateTypeDescriptor("RuntimeError"),
				Err:        fmt.Sprintf("Runtime panic: %s", errMsg),
			}}}
		}
	}
}
