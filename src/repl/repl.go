package repl

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/evaluator"
	"dash-lang.io/src/lexer"
	"dash-lang.io/src/parser"
	"dash-lang.io/src/semantic"
)

const WELCOME = `
██████╗  █████╗ ███████╗██╗  ██╗
██╔══██╗██╔══██╗██╔════╝██║  ██║
██║  ██║███████║███████╗███████║
██║  ██║██╔══██║╚════██║██╔══██║
██████╔╝██║  ██║███████║██║  ██║
╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝

Welcome to the Dash REPL - v0.0.1
`

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	fmt.Fprintf(out, "%s", WELCOME)

	var lines strings.Builder
	for {
		fmt.Fprintf(out, "%s", PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := strings.TrimSpace(scanner.Text())
		lcfg := &lexer.Config{SkipComments: true}
		switch line {
		case "exit":
			return
		case "clear":
			lines.Reset()
			continue
		case "history":
			fmt.Fprintln(out, lines.String())
			continue
		case "program":
			l := lexer.New("", lines.String(), lcfg)
			p := parser.New(l)
			fmt.Fprintln(out, p.ParseREPL().String())
			continue
		}

		l := lexer.New("", lines.String()+line, lcfg)
		p := parser.New(l)

		lib := p.ParseREPL()
		// remove main fn
		script := &ast.Library{Token: lib.Token, Name: lib.Name}
		for _, n := range lib.Nodes {
			switch n := n.(type) {
			case *ast.FunctionExpression:
				if n.Name.Value != "main" {
					script.Nodes = append(script.Nodes, n)
					continue
				}
				for _, stmt := range n.Body.Statements {
					script.Nodes = append(script.Nodes, stmt)
				}
			default:
				script.Nodes = append(script.Nodes, n)
			}
		}

		if len(p.Errors()) != 0 {
			fmt.Fprintln(out, strings.Join(p.Errors(), "\n"))
		}

		s := semantic.New("REPL", nil)
		s.Analyse(script)
		if len(s.Errors()) != 0 {
			fmt.Fprintln(out, strings.Join(s.Errors(), "\n"))
			continue
		}

		e := evaluator.New(nil)
		val := e.Eval(script, evaluator.NewContext(nil))
		switch val := val.(type) {
		case *evaluator.Return:
			if len(val.Values) == 0 {
				break
			}
			// check if error
			switch val := val.Values[0].(type) {
			case *evaluator.Error:
				fmt.Fprintf(out, "[ERROR] %v\n", val.Err)
				continue
			}
			fmt.Fprintf(out, "%v\n", val.Values...)
		default:
			fmt.Fprintf(out, "%v\n", val)

		}

		// only add line if no errors
		lines.WriteString(line + "\n")
	}
}
