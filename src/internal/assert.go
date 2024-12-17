package internal

import "fmt"

func AssertNotType[T any](v any, reason string) {
	if _, ok := v.(T); ok {
		msg := fmt.Sprintf("assertion failed: type was %T: ", v)
		panic(msg + reason)
	}
}

func AssertTrue(b bool, reason string) {
	if !b {
		panic("assertion failed: " + reason)
	}
}

func AssertNotNil(v any, reason string) {
	if v == nil {
		panic("assertion failed: " + reason)
	}
}
