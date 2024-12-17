package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"

	"dash-lang.io/src/builder"
	"dash-lang.io/src/repl"
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

	runCmd = flag.NewFlagSet("run", flag.ExitOnError)
)

func main() {
	if len(os.Args) < 2 {
		println("Expected command: play, build, or run")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("dash v%s %s/%s\n", version, osName, archName)
	case "play":
		repl.Start(os.Stdin, os.Stdout)
	case "build":
		buildCmd.Parse(os.Args[2:])
		if buildCmd.NArg() < 1 {
			fmt.Println("expected source file")
			buildCmd.PrintDefaults()
			os.Exit(1)
		}
		sourceFile := buildCmd.Arg(0)

		tmpDir, err := os.MkdirTemp("", "*")
		if err != nil {
			fmt.Println("unable to create temporary directory:", err)
			os.Exit(1)
		}
		cfg := &builder.Config{
			SrcDir:  sourceFile,
			WorkDir: tmpDir,
			OutDir:  ".",
		}
		if err := builder.Build(cfg); err != nil {
			fmt.Println("unable to build:", err)
			os.Exit(1)
		}
	case "run":
		buildCmd.Parse(os.Args[2:])
		if buildCmd.NArg() < 1 {
			fmt.Println("expected source file")
			buildCmd.PrintDefaults()
			os.Exit(1)
		}
		sourceFile := buildCmd.Arg(0)
		tmpDir, err := os.MkdirTemp("", "*")
		if err != nil {
			fmt.Println("unable to create temporary directory:", err)
			os.Exit(1)
		}

		cfg := &builder.Config{
			SrcDir:  sourceFile,
			WorkDir: tmpDir,
			OutDir:  tmpDir,
		}
		if err := builder.Build(cfg); err != nil {
			fmt.Println("unable to build:", err)
			os.Exit(1)
		}
		execPath := tmpDir + "/" + sourceFile[:len(sourceFile)-3]
		if err := syscall.Exec(execPath, []string{execPath}, os.Environ()); err != nil {
			fmt.Println("unable to run executable:", err)
			os.Exit(1)
		}
	default:
		println("Unknown command. Use: play, build, run")
		os.Exit(1)
	}
}
