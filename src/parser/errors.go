package parser

import "fmt"

// General errors
var (
	errInvalidToken = func(got string) error {
		return fmt.Errorf("invalid token '%s'", got)
	}
	errMissingArgumentType = func(argName string) error {
		return fmt.Errorf("argument '%s' missing type", argName)
	}
	errInvalidEscapeSequence = func(ch byte) error {
		return fmt.Errorf("invalid escape sequence '\\%c' in string literal", ch)
	}
)

// Attribute errors
var (
	errInvalidAttributeArgument = func(annotation, attribute string) error {
		return fmt.Errorf("invalid argument '%s' for attribute @%s", attribute, annotation)
	}
)

var (
	errMissingGenericConstraint = func(name string) error {
		return fmt.Errorf("generic constraint for type '%s' missing", name)
	}
)
