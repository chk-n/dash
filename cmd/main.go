package main

import (
	"flag"
	"fmt"
	"os"

	"dash-lang.io/src/builder"
	"dash-lang.io/src/evaluator"
	"dash-lang.io/src/repl"
	"dash-lang.io/src/tester"
	// "dash-lang.io/src/repl"
)

var (
	version  string
	osName   string
	archName string
)

var (
	playCmd     = flag.NewFlagSet("play", flag.ExitOnError)
	buildCmd    = flag.NewFlagSet("build", flag.ExitOnError)
	buildOutput = buildCmd.String("o", "main", "output file name")

	testCmd       = flag.NewFlagSet("test", flag.ExitOnError)
	testRecursive = testCmd.Bool("r", false, "recursively run tests in subdirectories")

	runCmd = flag.NewFlagSet("run", flag.ExitOnError)
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Expected command: play, build, or run")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("dash v%s %s/%s\n", version, osName, archName)
	case "play":
		repl.Start(os.Stdin, os.Stdout)
	case "test":
		testCmd.Parse(os.Args[2:])
		dir := "."
		if testCmd.NArg() > 0 {
			dir = testCmd.Arg(0)
		}
		if *testRecursive {
			tr := tester.NewTestRunner(dir)
			if err := tr.RunAll(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		} else {
			fmt.Println("non-recursive tests are currently not supported")
			os.Exit(1)
		}
	case "build":
		fmt.Println("'build' is currently not supported")
		os.Exit(1)
	case "run":
		buildCmd.Parse(os.Args[2:])
		if buildCmd.NArg() < 1 {
			fmt.Println("expected source file")
			buildCmd.PrintDefaults()
			os.Exit(1)
		}
		cfg := &builder.Config{
			SrcDir: buildCmd.Arg(0),
		}
		b := builder.New(cfg)
		libs, err := b.BuildProject()
		if err != nil {
			fmt.Println("unable to build:", err)
			os.Exit(1)
		}
		if err := evaluator.Execute(libs); err != nil {
			fmt.Println("unable to evaluate:", err)
			os.Exit(1)
		}
		// buildCmd.Parse(os.Args[2:])
		// if buildCmd.NArg() < 1 {
		// 	fmt.Println("expected source file")
		// 	buildCmd.PrintDefaults()
		// 	os.Exit(1)
		// }
		// sourceFileOrDir := buildCmd.Arg(0)
		// // tmpDir, err := os.MkdirTemp("", "*")
		// // if err != nil {
		// // 	fmt.Println("unable to create temporary directory:", err)
		// // 	os.Exit(1)
		// // }

		// cfg := &builder.Config{
		// 	SrcDir: sourceFileOrDir,
		// 	// WorkDir: tmpDir,
		// 	// OutDir:  tmpDir,
		// }
		// if err := builder.Build(cfg); err != nil {
		// 	fmt.Println("unable to build:", err)
		// 	os.Exit(1)
		// }
		// execPath := tmpDir + "/" + sourceFileOrDir[:len(sourceFileOrDir)-3]
		// if err := syscall.Exec(execPath, []string{execPath}, os.Environ()); err != nil {
		// 	fmt.Println("unable to run executable:", err)
		// 	os.Exit(1)
		// }
	default:
		fmt.Println("Unknown command. Use: play, build, run")
		os.Exit(1)
	}
}
