package builder

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/lexer"
	"dash-lang.io/src/merger"
	"dash-lang.io/src/parser"
	"dash-lang.io/src/semantic"
	"dash-lang.io/src/types"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	// Directory to evaluate
	SrcDir string
}

type Builder struct {
	srcDir string
	// root directory of project
	projectRoot string
	// name of project as defined
	// within dash.toml
	projectName string
}

func New(cfg *Config) *Builder {
	return &Builder{srcDir: cfg.SrcDir}
}

// BUG: if srcDir is "." then we cant build
func (b *Builder) BuildProject() (_ map[string]*ast.Library, err error) {
	if b.projectRoot, err = findProjectRoot(b.srcDir); err != nil {
		return nil, err
	}

	filesPerDir, err := b.findAllDashFiles()
	if err != nil {
		return nil, fmt.Errorf("unable to find dash files: %s", err)
	}

	if b.projectName, err = getProjectName(b.projectRoot); err != nil {
		return nil, err
	}

	root, err := b.buildDependencyTree(b.srcDir, filesPerDir)
	if err != nil {
		return nil, fmt.Errorf("unable to build dependency tree: %s", err)
	}

	libs, err := b.buildProject(root)
	if err != nil {
		return nil, err
	}

	return libs, nil
}

func (b *Builder) findAllDashFiles() (_ map[string][]string, err error) {
	// walk files in project
	filesPerDir := make(map[string][]string)
	if err := filepath.Walk(b.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// skip directories
		if info.IsDir() {
			return nil
		}

		// we only want .ds files
		if !strings.HasSuffix(path, ".ds") {
			return nil
		}

		dir := filepath.Dir(path)
		filesPerDir[dir] = append(filesPerDir[dir], path)

		return nil
	}); err != nil {
		return nil, err
	}

	return filesPerDir, nil
}

func (b *Builder) buildProject(root *dependencyTree) (map[string]*ast.Library, error) {
	libs := make(map[string]*ast.Library)
	var pErrs []string
	var sErrs []string

	// parseAndMerge parses an entire library and performs
	// semantical analysis on that library potentially
	// using externally visible vars, types and functions
	// from imported libs
	parseAndMerge := func(dep *dependencyTree) error {
		var lib *ast.Library

		// parse and merge all files within a directory
		for _, path := range dep.Files {
			file, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", path, err)
			}

			l := lexer.New(path, string(file), &lexer.Config{SkipComments: true})
			p := parser.New(l)
			libPart := p.ParseLibrary()

			errs := p.Errors()
			if len(errs) > 0 {
				pErrs = append(pErrs, errs...)
				continue
			}

			if lib != nil {
				lib = merger.Merge2(lib, libPart)
			} else {
				lib = libPart
			}

		}

		// NOTE: in future this will also contain the path to the lib
		libImportName := fmt.Sprintf("%s/%s", b.projectName, lib.Name.String())

		if _, ok := libs[libImportName]; ok {
			return fmt.Errorf("duplicate library name '%s' found in directory '%s'",
				lib.Name.String(), dep.AbsoluteDir)
		}

		// extract all global exported types from imported libs
		// and create a map for semsis
		typeTable := make(map[string]map[string]types.TypeSpec)
		for _, n := range lib.Nodes {
			switch n := n.(type) {
			case *ast.UseStatement:
				importName := n.Name.TokenLiteral()
				if _, ok := typeTable[importName]; ok {
					continue
				}

				importedLib, ok := libs[importName]
				if !ok {
					panic("unable to find libary " + importName)
				}
				typeTable[importName] = importedLib.Exports()
			}
		}

		// check for semantical errors
		s := semantic.New(dep.AbsoluteDir, typeTable)
		s.Analyse(lib)
		if len(s.Errors()) > 0 {
			sErrs = append(sErrs, s.ErrorsFmt()...)
		}

		libs[libImportName] = lib

		return nil
	}

	if err := walkDependencyTree(root, parseAndMerge); err != nil {
		return nil, fmt.Errorf("unable to walk dependency tree: %s", err)
	}

	if len(pErrs) > 0 && len(sErrs) > 0 {
		allErrs := make([]string, len(pErrs)+len(sErrs))
		copy(allErrs[:len(pErrs)], pErrs)
		copy(allErrs[len(pErrs):], sErrs)
		return nil, errors.New(strings.Join(allErrs, "\n"))
	} else if len(pErrs) > 0 {
		return nil, errors.New(strings.Join(pErrs, "\n"))
	} else if len(sErrs) > 0 {
		return nil, errors.New(strings.Join(sErrs, "\n"))
	}

	return libs, nil
}

func (b *Builder) buildDependencyTree(entryLib string, filesPerDir map[string][]string) (*dependencyTree, error) {
	// maps 'dir' to a DependecyTree node
	nodes := make(map[string]*dependencyTree, len(filesPerDir))
	// contains all imports made by a library using 'dir' as key
	importMap := make(map[string][]string)

	for dir, files := range filesPerDir {

		var libName string
		for _, file := range files {
			lib, err := extractImports(file)
			if err != nil {
				return nil, err
			}

			if libName == "" {
				libName = fmt.Sprintf("%s/%s", b.projectName, lib.Name.String())
			}
			// skip if already parsed
			if _, exists := importMap[libName]; exists {
				continue
			}

			for _, n := range lib.Nodes {
				switch n := n.(type) {
				case *ast.UseStatement:
					importName := n.Name.TokenLiteral()
					importMap[libName] = append(importMap[libName], importName)
				}
			}
		}
		if strings.Contains(dir, entryLib) {
			entryLib = libName
		}
		nodes[libName] = &dependencyTree{AbsoluteDir: dir, Files: files}
	}

	for dir, imps := range importMap {
		imports := []*dependencyTree{}
		for _, imp := range imps {
			child, ok := nodes[imp]
			if !ok {
				panic("empty")
			}
			imports = append(imports, child)
		}
		nodes[dir].Imports = imports
	}

	return nodes[entryLib], nil
}

// ------- //
// Helpers //
// ------- //

func findProjectRoot(startDir string) (string, error) {
	info, err := os.Stat(startDir)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		return "", fmt.Errorf("expected directory but got file")
	}
	// get absolute path
	absDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("unable to get absolute path: %s", err)
	}

	currentDir := absDir
	for {
		// return root dir if contains dash.toml
		dashTomlPath := filepath.Join(currentDir, "dash.toml")
		if _, err := os.Stat(dashTomlPath); err == nil {
			return currentDir, nil
		}

		// climb up until root
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("no dash.toml file found in directory tree from %s", startDir)
		}
		currentDir = parentDir
	}
}

func getProjectName(rootDir string) (string, error) {
	var doc struct {
		Library struct {
			Name    string
			Version string
		}
	}

	b, err := os.ReadFile(rootDir + "/dash.toml")
	if err != nil {
		return "", err
	}

	if err := toml.Unmarshal(b, &doc); err != nil {
		return "", err
	}

	return doc.Library.Name, nil
}

// extractImports parses a file to extract only the import statements
func extractImports(filePath string) (*ast.Library, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	l := lexer.New(filePath, string(file), &lexer.Config{SkipComments: true})
	p := parser.New(l)
	lib := p.ParseImports()

	return lib, nil
}

type dependencyTree struct {
	Visitied    bool
	AbsoluteDir string
	// files that make up the library
	Files []string
	// libraries imported by current library
	Imports []*dependencyTree
}

func walkDependencyTree(parent *dependencyTree, visit func(dep *dependencyTree) error) error {
	for _, child := range parent.Imports {
		if !child.Visitied {
			walkDependencyTree(child, visit)
		}
	}

	parent.Visitied = true
	return visit(parent)
}
