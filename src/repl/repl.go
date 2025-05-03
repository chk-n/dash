package repl

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"dash-lang.io/src/generator"
	"dash-lang.io/src/lexer"
	"dash-lang.io/src/parser"
	"dash-lang.io/src/semantic"
	// "dash-lang.io/src/transformer"
)

const WELCOME = `██████╗  █████╗ ███████╗██╗  ██╗
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

		script := p.ParseREPL()

		if len(p.Errors()) != 0 {
			for _, msg := range p.Errors() {
				fmt.Fprintf(out, "Parser error: %s\n", msg)
			}
			continue
		}

		s := semantic.New("", nil)
		s.Analyse(script)
		if len(s.Errors()) != 0 {
			fmt.Fprintf(out, "Semantic analysis error: %s", s.Errors())
		}

		// t := transformer.New()
		// t.Tranform(script)

		cfg := &generator.Config{
			// leave Triple nil to use system native
			Mode:      generator.REPL,
			ModuleTag: "repl",
		}
		c := generator.New(cfg)
		err := c.GenerateAndExec(script)
		if err != nil {
			fmt.Fprintf(out, "Compilation error: %s\n", err)
			continue
		}
		// only add line if no errors
		lines.WriteString(line + "\n")
	}
}
