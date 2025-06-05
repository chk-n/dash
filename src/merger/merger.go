package merger

import "dash-lang.io/src/ast"

func Merge2(a1, a2 *ast.Library) *ast.Library {
	if a1.Name.String() != a2.Name.String() {
		panic("can only merge ASTs from files within the same library")
	}

	m := &ast.Library{
		Token: a1.Token,
		Name:  a1.Name,
		Nodes: make([]ast.Node, len(a1.Nodes)+len(a2.Nodes)),
	}

	copy(m.Nodes, a1.Nodes)
	copy(m.Nodes[len(a1.Nodes):], a2.Nodes)

	return m
}
