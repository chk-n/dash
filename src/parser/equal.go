package parser

// import (
// 	"fmt"

// 	"dash-lang.io/srcast"
// 	"dash-lang.io/srctoken"
// )

// func ModuleStatementsEqual(ms1, ms2 *ast.ModuleStatement) bool {
// 	if ms1 == nil && ms2 == nil {
// 		return true
// 	}
// 	if ms1 == nil || ms2 == nil {
// 		return false
// 	}

// 	if !(TokenStatementEqual(ms1.Token, ms2.Token)) ||
// 		!ExpressionEqual(ms1.Name, ms2.Name) ||
// 		!ImportStatementsEqual(ms1.Imports, ms2.Imports) ||
// 		!StructStatementsEqual(ms1.Structs, ms2.Structs) ||
// 		!InterfaceStatementsEqual(ms1.Interfaces, ms2.Interfaces) ||
// 		!FunctionLiteralEqual(ms1.Functions, ms2.Functions) {
// 		return false
// 	}

// 	return true
// }

// func ImportStatementsEqual(is1, is2 []ast.ImportStatement) bool {

// 	if len(is1) != len(is2) {
// 		fmt.Println("ImportStatements not same length")
// 		return false
// 	}

// 	for i := 0; i < len(is1); i++ {
// 		if !TokenStatementEqual(is1[i].Token, is2[i].Token) {
// 			return false
// 		}

// 		if !TokenStatementEqual(is1[i].Package, is2[i].Package) {
// 			return false
// 		}
// 	}

// 	return true
// }

// func StructStatementsEqual(ss1, ss2 []ast.StructStatement) bool {
// 	if len(ss1) != len(ss2) {
// 		fmt.Printf("SchemaStatements not equal: %d != %d\n", len(ss1), len(ss2))
// 		return false
// 	}

// 	for i := 0; i < len(ss1); i++ {
// 		if ss1[i].Public != ss2[i].Public {
// 			return false
// 		}
// 		if !IdentifierStatementsEqual(ss1[i].Name, ss2[i].Name) {
// 			return false
// 		}

// 		if !TokenStatementEqual(ss1[i].Token, ss2[i].Token) {
// 			return false
// 		}

// 		if !AnnotationStatementEqual(ss1[i].Annotations, ss2[i].Annotations) {
// 			return false
// 		}

// 		if !StructFieldStatementsEqual(ss1[i].Fields, ss2[i].Fields) {
// 			return false
// 		}
// 	}
// 	return true
// }

// func StructFieldStatementsEqual(sfs1, sfs2 []*ast.StructFieldStatement) bool {
// 	if len(sfs1) != len(sfs2) {
// 		return false
// 	}

// 	for i := 0; i < len(sfs1); i++ {
// 		if sfs1[i] == nil && sfs2[i] == nil {
// 			return true
// 		}
// 		if sfs1[i] == nil || sfs2[i] == nil {
// 			return false
// 		}
// 		if !IdentifierStatementsEqual(sfs1[i].Name, sfs2[i].Name) {
// 			return false
// 		}

// 		if !ValueTypeStatementEqual(sfs1[i].Type, sfs2[i].Type) {
// 			return false
// 		}
// 	}

// 	return true
// }

// func InterfaceStatementsEqual(if1, if2 []ast.InterfaceStatement) bool {
// 	if len(if1) != len(if2) {
// 		return false
// 	}

// 	for i := 0; i < len(if1); i++ {
// 		if if1[i].Public != if2[i].Public {
// 			return false
// 		}

// 		if !IdentifierStatementsEqual(if1[i].Name, if2[i].Name) {
// 			return false
// 		}
// 		if !FunctionSignatureStatementsEqual(if1[i].Functions, if2[i].Functions) {
// 			return false
// 		}

// 		if !TokenStatementEqual(if1[i].Token, if2[i].Token) {
// 			return false
// 		}
// 	}
// 	return true
// }

// func FunctionLiteralEqual(fs1, fs2 []ast.FunctionLiteral) bool {
// 	if len(fs1) != len(fs2) {
// 		return false
// 	}
// 	if fs1 == nil && fs2 == nil {
// 		return true
// 	}
// 	if fs1 == nil || fs2 == nil {
// 		return false
// 	}

// 	for i := 0; i < len(fs1); i++ {
// 		if !TokenStatementEqual(fs1[i].Token, fs2[i].Token) {
// 			return false
// 		}
// 		if !FunctionSignatureStatementEqual(*fs1[i].Signature, *fs2[i].Signature) {
// 			return false
// 		}
// 		if !BlockStatementsEqual(fs1[i].Body, fs2[i].Body) {
// 			return false
// 		}
// 	}
// 	// TODO: Implement function statement comparison
// 	return true
// }

// func FunctionSignatureStatementsEqual(fss1, fss2 []ast.FunctionSignatureStatement) bool {
// 	if len(fss1) != len(fss2) {
// 		return false
// 	}

// 	for i := 0; i < len(fss1); i++ {
// 		if !FunctionSignatureStatementEqual(fss1[i], fss2[i]) {
// 			return false
// 		}
// 	}

// 	return true
// }

// func FunctionSignatureStatementEqual(fss1, fss2 ast.FunctionSignatureStatement) bool {
// 	if !IdentifierStatementsEqual(fss1.Name, fss2.Name) {
// 		return false
// 	}
// 	if !ParameterStatementsEqual(fss1.ParamValues, fss2.ParamValues) {
// 		return false
// 	}

// 	if len(fss1.ReturnValues) != len(fss2.ReturnValues) {
// 		return false
// 	}
// 	return true
// }

// func ParameterStatementsEqual(ps1, ps2 []ast.ParameterStatement) bool {
// 	if len(ps1) != len(ps2) {
// 		return false
// 	}

// 	for i := 0; i < len(ps1); i++ {
// 		if !IdentifierStatementsEqual(ps1[i].Name, ps2[i].Name) {
// 			return false
// 		}
// 		if !ValueTypeStatementEqual(ps1[i].Type, ps2[i].Type) {
// 			return false
// 		}
// 		if !ExpressionEqual(ps1[i].OptionalValue, ps2[i].OptionalValue) {
// 			return false
// 		}
// 	}

// 	return true
// }

// func IfStatementsEqual(if1, if2 *ast.IfStatement) bool {
// 	if if1 == nil && if2 == nil {
// 		return true
// 	}
// 	if if1 == nil || if2 == nil {
// 		return false
// 	}

// 	if !TokenStatementEqual(if1.Token, if2.Token) {
// 		return false
// 	}

// 	if !ExpressionEqual(if1.Condition, if2.Condition) {
// 		return false
// 	}

// 	if !BlockStatementsEqual(if1.Consequence, if2.Consequence) {
// 		return false
// 	}

// 	if !ElIfStatementsEqual(if1.Alternative, if2.Alternative) {
// 		return false
// 	}

// 	if !BlockStatementsEqual(if1.FinalAlternative, if2.FinalAlternative) {
// 		return false
// 	}

// 	return true
// }

// func ElIfStatementsEqual(eif1, eif2 *ast.ElIfStatement) bool {
// 	if eif1 == nil && eif2 == nil {
// 		return true
// 	}
// 	if eif1 == nil || eif2 == nil {
// 		return false
// 	}

// 	if !TokenStatementEqual(eif1.Token, eif2.Token) {
// 		return false
// 	}

// 	if !ExpressionEqual(eif1.Condition, eif2.Condition) {
// 		return false
// 	}

// 	if !BlockStatementsEqual(eif1.Consequence, eif2.Consequence) {
// 		return false
// 	}

// 	if !ElIfStatementsEqual(eif1.Alternative, eif2.Alternative) {
// 		return false
// 	}

// 	return true
// }

// func BlockStatementsEqual(bs1, bs2 *ast.BlockStatement) bool {
// 	if bs1 == nil && bs2 == nil {
// 		return true
// 	}
// 	if bs1 == nil || bs2 == nil {
// 		return false
// 	}
// 	panic("Not implemented")
// }

// func IdentifierStatementsEqual(id1, id2 *ast.Identifier) bool {
// 	if id1 == nil && id2 == nil {
// 		return true
// 	}
// 	if id1 == nil || id2 == nil {
// 		return false
// 	}

// 	if !TokenStatementEqual(id1.Token, id2.Token) {
// 		return false
// 	}
// 	if id1.Value != id2.Value {
// 		fmt.Printf("Identifier value mismatch: %s != %s\n", id1.Value, id2.Value)
// 		return false
// 	}

// 	return true
// }

// func ReturnStatementEqual(rs1, rs2 *ast.ReturnStatement) bool {
// 	if rs1 == nil && rs2 == nil {
// 		return true
// 	}
// 	if rs1 == nil || rs2 == nil {
// 		return false
// 	}

// 	if !TokenStatementEqual(rs1.Token, rs2.Token) {
// 		return false
// 	}

// 	if len(rs1.ReturnValues) != len(rs2.ReturnValues) {
// 		return false
// 	}

// 	for i := 0; i < len(rs1.ReturnValues); i++ {
// 		if !ExpressionEqual(rs1.ReturnValues[i], rs2.ReturnValues[i]) {
// 			return false
// 		}
// 	}

// 	return true
// }

// func ListTypeStatementEqual(ls1, ls2 *ast.ListTypeStatement) bool {
// 	if ls1 == nil && ls2 == nil {
// 		return true
// 	}
// 	if ls1 == nil || ls2 == nil {
// 		return false
// 	}

// 	if !ExpressionEqual(ls1.Size, ls2.Size) {
// 		return false
// 	}

// 	if !TokenStatementEqual(ls1.Token, ls2.Token) {
// 		return false
// 	}

// 	if ls1.Type != nil && ls2.Type != nil && !ValueTypeStatementEqual(*ls1.Type, *ls2.Type) {
// 		return false
// 	}

// 	return ListTypeStatementEqual(ls1.Inner, ls2.Inner)
// }

// func SetTypeStatementEqual(ls1, ls2 *ast.SetTypeStatement) bool {
// 	if ls1 == nil && ls2 == nil {
// 		return true
// 	}
// 	if ls1 == nil || ls2 == nil {
// 		return false
// 	}

// 	if !ExpressionEqual(ls1.Size, ls2.Size) {
// 		return false
// 	}

// 	if !TokenStatementEqual(ls1.Token, ls2.Token) {
// 		return false
// 	}

// 	if ls1.Type != nil && ls2.Type != nil && !ValueTypeStatementEqual(*ls1.Type, *ls2.Type) {
// 		return false
// 	}

// 	return SetTypeStatementEqual(ls1.Inner, ls2.Inner)
// }

// func AnnotationStatementEqual(as1, as2 []*ast.AnnotationStatement) bool {
// 	if as1 == nil && as2 == nil {
// 		return true
// 	}
// 	if as1 == nil || as2 == nil {
// 		return false
// 	}

// 	if len(as1) != len(as2) {
// 		return false
// 	}

// 	for i := 0; i < len(as1); i++ {
// 		if !TokenStatementEqual(as1[i].Token, as2[i].Token) {
// 			return false
// 		}

// 		for j := 0; j < len(as1); j++ {
// 			if len(as1[i].Names) != len(as2[i].Names) {
// 				return false
// 			}
// 			if !IdentifierStatementsEqual(as1[i].Names[j], as2[i].Names[j]) {
// 				return false
// 			}
// 		}
// 	}

// 	return true
// }

// func ExpressionEqual(exp1, exp2 ast.Expression) bool {
// 	if exp1 == nil && exp2 == nil {
// 		return true
// 	}
// 	if exp1 == nil || exp2 == nil {
// 		return false
// 	}
// 	if exp1.String() != exp2.String() {
// 		fmt.Printf("Expression mismatch: %s != %s\n", exp1.String(), exp2.String())
// 		return false
// 	}

// 	if exp1.TokenLiteral() != exp2.TokenLiteral() {
// 		fmt.Printf("TokenLiteral mismatch: %s != %s\n", exp1.TokenLiteral(), exp2.TokenLiteral())
// 		return false
// 	}
// 	return true
// }

// func PrefixExpressionsEqual(pe1, pe2 *ast.PrefixExpression) bool {
// 	if pe1 == nil && pe2 == nil {
// 		return true
// 	}
// 	if pe1 == nil || pe2 == nil {
// 		return false
// 	}

// 	if !TokenStatementEqual(pe1.Token, pe2.Token) {
// 		return false
// 	}

// 	if pe1.Operator != pe2.Operator {
// 		return false
// 	}

// 	if !ExpressionEqual(pe1.Right, pe2.Right) {
// 		return false
// 	}

// 	return true
// }

// func ParameterStatementEqual(ps1, ps2 ast.ParameterStatement) bool {
// 	if !IdentifierStatementsEqual(ps1.Name, ps2.Name) {
// 		return false
// 	}

// 	// if !IdentifierStatementsEqual(ps1.OptionalValue, ps2.OptionalValue) {
// 	// 	return false
// 	// }

// 	if !ValueTypeStatementEqual(ps1.Type, ps2.Type) {
// 		return false
// 	}

// 	return true
// }

// func ValueTypeStatementEqual(vt1, vt2 ast.ValueTypeStatement) bool {
// 	return TokenStatementEqual(vt1.Token, vt2.Token)
// }

// func TokenStatementEqual(ts1, ts2 token.Token) bool {
// 	if ts1.Literal != ts2.Literal {
// 		fmt.Printf("Literal mismatch: %s != %s\n", ts1.Literal, ts2.Literal)
// 		return false
// 	}
// 	if string(ts1.Type) != string(ts2.Type) {
// 		fmt.Printf("Type mismatch: %s != %s\n", ts1.Type, ts2.Type)
// 		return false
// 	}
// 	if ts1.Line != ts2.Line {
// 		fmt.Printf("Line mismatch: %d != %d\n", ts1.Line, ts2.Line)
// 		return false
// 	}

// 	return true
// }
