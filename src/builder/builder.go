package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/lexer"
	"dash-lang.io/src/merger"
	"dash-lang.io/src/parser"
	"dash-lang.io/src/semantic"
)

type Config struct {
	// file or directory to compile
	SrcDir string

	// Default: DASH_HOME env variable
	DashHomeDir string
}

func BuildProject(cfg *Config) (map[string]*ast.Library, error) {
	// Create a map to store files grouped by their immediate directory
	filesPerDir := make(map[string][]string)

	// Walk through all directories recursively
	err := filepath.Walk(cfg.SrcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories themselves
		if info.IsDir() {
			return nil
		}

		// Only process .ds files
		if !strings.HasSuffix(path, ".ds") {
			return nil
		}

		// Get the directory path for this file
		dir := filepath.Dir(path)

		// Add the file to its directory group
		filesPerDir[dir] = append(filesPerDir[dir], path)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("unable to get all .ds files: %w", err)
	}

	libs := make(map[string]*ast.Library)
	// Process each directory's files
	for dir, files := range filesPerDir {
		var lib *ast.Library

		// merge all partial libraries in same dir into one lib
		for _, path := range files {
			file, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", path, err)
			}

			l := lexer.New(path, string(file), &lexer.Config{})
			p := parser.New(l)
			libPart := p.ParseLibrary()

			if libPart == nil {
				return nil, fmt.Errorf("failed to parse library from file %s", path)
			}

			if lib != nil {
				lib = merger.Merge2(lib, libPart)
			} else {
				lib = libPart
			}

		}

		if lib == nil {
			continue // Skip directories with no valid libraries
		}

		s := semantic.New()
		s.Analyse(lib)
		if _, ok := libs[lib.Name.String()]; ok {
			return nil, fmt.Errorf("duplicate library name '%s' found in directory '%s'",
				lib.Name.String(), dir)
		}

		libs[lib.Name.String()] = lib
	}

	return libs, nil
}

// ------- //
// Helpers //
// ------- //

func getAllFiles(currentDir string) ([][]string, error) {
	// walk directories recursively and parse
	// all .ds files and merge into one library
	dirs := []string{currentDir}
	filesPerDirectory := [][]string{}

	for len(dirs) > 0 {
		dir := dirs[0]
		dirs = dirs[1:]

		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}

		files := []string{}
		for _, entry := range entries {
			path := dir + "/" + entry.Name()

			if entry.IsDir() {
				dirs = append(dirs, path)
			} else if strings.HasSuffix(entry.Name(), ".ds") {
				files = append(files, path)
			}
		}
		if len(files) == 0 {
			continue
		}
		filesPerDirectory = append(filesPerDirectory, files)
	}

	return filesPerDirectory, nil
}

// import (
// 	"bytes"
// 	"crypto/sha256"
// 	"encoding/hex"
// 	"fmt"
// 	"os"
// 	"os/exec"
// 	"strings"
// 	"unicode"

// 	"dash-lang.io/src/generator"
// 	"dash-lang.io/src/lexer"
// 	"dash-lang.io/src/parser"
// 	"dash-lang.io/src/semantic"
// 	"dash-lang.io/src/transformer"
// 	"tinygo.org/x/go-llvm"
// )

// type Config struct {
// 	// file or directory to compile
// 	SrcDir string
// 	// the name of the final executable
// 	// Default: main
// 	ExecutableName string
// 	// Set compilation directory, leave empty
// 	// to use the current working directory
// 	WorkDir string
// 	// where final executable should live
// 	OutDir string
// 	// If set to true it will print build steps
// 	// Debug  bool
// 	Triple string
// 	// where dash standard library and runtime files live
// 	// Default: DASH_HOME env variable
// 	DashHomeDir string
// }

// // Emits executable in chosen directory
// func Build(cfg *Config) error {
// 	if strings.HasSuffix(cfg.SrcDir, ".ds") {
// 		if cfg.ExecutableName == "" {
// 			cfg.ExecutableName = strings.Split(cfg.SrcDir, ".")[0]
// 		}
// 	} else {
// 		return fmt.Errorf("compiling directories currently not supported")
// 	}

// 	if cfg.DashHomeDir == "" {
// 		cfg.DashHomeDir = os.Getenv("DASH_HOME")
// 	}

// 	if cfg.ExecutableName == "" {
// 		cfg.ExecutableName = "main"
// 	}
// 	genCfg := &generator.Config{}
// 	machine, err := generator.NewTargetMachine(genCfg)
// 	if err != nil {
// 		return err
// 	}

// 	if cfg.Triple == "" {
// 		cfg.Triple = machine.Triple()
// 	}

// 	var paths []string
// 	path, err := buidLibrary(cfg, cfg.SrcDir)
// 	if err != nil {
// 		return err
// 	}
// 	// manually add runtime
// 	paths = append(paths, path)
// 	path, err = buidLibrary(cfg, cfg.DashHomeDir+"/src/runtime/runtime.ds")
// 	if err != nil {
// 		return err
// 	}
// 	paths = append(paths, path)

// 	// Link modules together
// 	ctx := llvm.NewContext()
// 	mainMod := ctx.NewModule("main")
// 	mainMod.SetTarget(machine.Triple())
// 	defer func() {
// 		if !mainMod.IsNil() {
// 			mainMod.Dispose()
// 			ctx.Dispose()
// 		}
// 	}()

// 	fmt.Println("compiling for:", cfg.Triple)
// 	for _, path := range paths {
// 		mod, err := ctx.ParseBitcodeFile(path)

// 		fmt.Println("linking modules:", path)
// 		err = llvm.LinkModules(mainMod, mod)
// 		if err != nil {
// 			return fmt.Errorf("unable to link module: %s", err)
// 		}
// 	}

// 	if err := llvm.VerifyModule(mainMod, llvm.ReturnStatusAction); err != nil {
// 		return fmt.Errorf("verify module failed: %s", err)
// 	}

// 	// Generate object file
// 	objFile := cfg.WorkDir + "/" + cfg.ExecutableName + ".o"
// 	llvmBuf, err := machine.EmitToMemoryBuffer(mainMod, llvm.ObjectFile)
// 	if err != nil {
// 		return err
// 	}

// 	defer llvmBuf.Dispose()
// 	if err := os.WriteFile(objFile, llvmBuf.Bytes(), 0666); err != nil {
// 		return err
// 	}

// 	osBase := getOS(cfg.Triple)
// 	switch osBase {
// 	case "darwin":
// 		linkDarwin(cfg)
// 	// case "linux":
// 	// linkLinux(cfg)
// 	default:
// 		return fmt.Errorf("unsupported os: %s", osBase)
// 	}

// 	return nil
// }

// func buidLibrary(cfg *Config, dir string) (string, error) {
// 	// Use Glob to find all dash files in the directory
// 	// pattern := filepath.Join(dir, "*.ds")
// 	// files, err := filepath.Glob(pattern)
// 	// if err != nil {
// 	// 	panic("error while finding dash files: " + err.Error())
// 	// }

// 	file := dir
// 	// asts := make([]*ast.Library, len(files))
// 	// for i, file := range files {

// 	srcBytes, err := os.ReadFile(file)
// 	if err != nil {
// 		return "", fmt.Errorf("unable to read file: " + file)
// 	}
// 	src := string(srcBytes)
// 	lcfg := &lexer.Config{SkipComments: true}
// 	l := lexer.New(dir, src, lcfg)

// 	p := parser.New(l)
// 	asts := p.ParseLibrary()
// 	// }

// 	// ast := merger.Merge2(asts[])

// 	s := semantic.New()
// 	s.Analyse(asts)
// 	errs := s.Errors()
// 	if len(errs) != 0 {
// 		return "", fmt.Errorf("%s", strings.Join(errs, "\n"))
// 	}

// 	t := transformer.New()
// 	t.Tranform(asts)

// 	genCfg := &generator.Config{
// 		// Mode: generator.DEBUG,
// 	}
// 	g := generator.New(genCfg)
// 	mod, err := g.GenerateLibrary(asts)
// 	if err != nil {
// 		return "", err
// 	}
// 	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
// 		return "", err
// 	}

// 	// Hash the file
// 	sum := sha256.Sum256(srcBytes)
// 	hexSum := hex.EncodeToString(sum[:16])

// 	f, err := os.Create(cfg.WorkDir + "/tmp_" + hexSum + ".bc")
// 	if err != nil {
// 		return "", err
// 	}

// 	err = llvm.WriteBitcodeToFile(mod, f)

// 	err = f.Close()
// 	if err != nil {
// 		return "", err
// 	}
// 	return f.Name(), nil
// }

// // return underlying OS without version (if contained) from a triple
// func getOS(triple string) string {
// 	osName := strings.Split(triple, "-")[2]
// 	return strings.TrimRightFunc(osName, func(r rune) bool {
// 		return unicode.IsNumber(r) || r == '.'
// 	})
// }

// // -------------- //
// // System Linkers //
// // -------------- //

// func linkDarwin(cfg *Config) error {
// 	outFile := cfg.OutDir + "/" + cfg.ExecutableName
// 	objFile := cfg.WorkDir + "/" + cfg.ExecutableName + ".o"
// 	arch := strings.Split(cfg.Triple, "-")[0]

// 	cmd := exec.Command("ld", []string{
// 		cfg.DashHomeDir + "/lib/libSystem-macos-" + arch + ".dylib",
// 		"-arch", arch,
// 		"-platform_version", "macos", "13.0", "13.0",
// 		"-o", outFile,
// 		objFile}...,
// 	)
// 	var buf bytes.Buffer
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("unable to run linker: %s: %s", err, buf.String())
// 	}
// 	return nil
// }

// func linkLinux(cfg *Config) error {
// 	outFile := cfg.OutDir + "/" + cfg.ExecutableName
// 	objFile := cfg.WorkDir + "/" + cfg.ExecutableName + ".o"

// 	cmd := exec.Command("ld", []string{
// 		"-z", "noexecstack",
// 		"-o", outFile,
// 		objFile,
// 		"-dynamic-linker", "/lib64/ld-linux-x86-64.so.2",
// 		"-L/usr/lib",
// 		"-lc",
// 		"-e", "main"}...,
// 	)
// 	var buf bytes.Buffer
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("unable to run linker: %s: %s", err, buf.String())
// 	}

// 	return nil
// }
