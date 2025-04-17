package merger

import "dash-lang.io/src/ast"

func Merge2(a1, a2 *ast.Library) *ast.Library {
	if a1.Name.String() != a2.Name.String() {
		panic("can only merge ASTs from files within the same library")
	}

	m := &ast.Library{
		Token:           a1.Token,
		Name:            a1.Name,
		Imports:         make([]*ast.UseStatement, len(a1.Imports)+len(a2.Imports)),
		TypeDefinitions: make([]*ast.TypeDefinitionStatement, len(a1.TypeDefinitions)+len(a2.TypeDefinitions)),
		TypeAliases:     make([]*ast.TypeAliasStatement, len(a1.TypeAliases)+len(a2.TypeAliases)),
		Structs:         make([]*ast.StructStatement, len(a1.Structs)+len(a2.Structs)),
		Enums:           make([]*ast.EnumStatement, len(a1.Enums)+len(a2.Enums)),
		Unions:          make([]*ast.UnionStatement, len(a1.Unions)+len(a2.Unions)),
		GlobalVariables: make([]*ast.AssignmentStatement, len(a1.GlobalVariables)+len(a2.GlobalVariables)),
		Errors:          make([]*ast.ErrorStatement, len(a1.Errors)+len(a2.Errors)),
		Functions:       make([]*ast.FunctionExpression, len(a1.Functions)+len(a2.Functions)),
	}

	// Populate Imports field
	copy(m.Imports, a1.Imports)
	copy(m.Imports[len(a1.Imports):], a2.Imports)

	// Populate TypeDefinitions field
	copy(m.TypeDefinitions, a1.TypeDefinitions)
	copy(m.TypeDefinitions[len(a1.TypeDefinitions):], a2.TypeDefinitions)

	// Populate TypeAliases field
	copy(m.TypeAliases, a1.TypeAliases)
	copy(m.TypeAliases[len(a1.TypeAliases):], a2.TypeAliases)

	// Populate Structs field
	copy(m.Structs, a1.Structs)
	copy(m.Structs[len(a1.Structs):], a2.Structs)

	// Populate Enums field
	copy(m.Enums, a1.Enums)
	copy(m.Enums[len(a1.Enums):], a2.Enums)

	// Populate Unions field
	copy(m.Unions, a1.Unions)
	copy(m.Unions[len(a1.Unions):], a2.Unions)

	// Populate GlobalVariables field
	copy(m.GlobalVariables, a1.GlobalVariables)
	copy(m.GlobalVariables[len(a1.GlobalVariables):], a2.GlobalVariables)

	// Populate Errors field
	copy(m.Errors, a1.Errors)
	copy(m.Errors[len(a1.Errors):], a2.Errors)

	// Populate Functions field
	copy(m.Functions, a1.Functions)
	copy(m.Functions[len(a1.Functions):], a2.Functions)

	return m
}
