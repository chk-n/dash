package builder

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/internal"
	"dash-lang.io/src/lexer"
	"dash-lang.io/src/merger"
	"dash-lang.io/src/parser"
	"dash-lang.io/src/semantic"
	"dash-lang.io/src/types"
)

type Config struct {
	// file or directory to compile
	SrcDir string

	IncludeStdLib bool
	// Default: DASH_HOME env variable
	DashHomeDir string
}

func BuildProject(cfg *Config) (map[string]*ast.Library, error) {
	filesPerDir := make(map[string][]string)

	projectRoot, err := findProjectRoot(cfg.SrcDir)
	if err != nil {
		return nil, err
	}

	// walk files in project
	if err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
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
		return nil, fmt.Errorf("unable to get all .ds files: %w", err)
	}

	var pErrs []string
	var sErrs []string
	libs := make(map[string]*ast.Library)

	// parse parses an entire library and performs
	// semantical analysis on that library potentially
	// using externally visible vars, types and functions
	// from imported libs
	parse := func(dep *dependencyTree) error {
		var lib *ast.Library

		// TODO: keep track of parse errors for each library
		// merge all partial libraries in same dir into one lib
		for _, path := range dep.Files {
			file, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", path, err)
			}

			l := lexer.New(path, string(file), &lexer.Config{SkipComments: true})
			p := parser.New(l)
			libPart := p.ParseLibrary()

			// TODO: if there was a problem track errors and continue
			if libPart == nil {
				pErrs = append(pErrs, p.Errors()...)
				continue
			}

			if lib != nil {
				lib = merger.Merge2(lib, libPart)
			} else {
				lib = libPart
			}

		}

		// extract all global exported types from imported libs
		// and create a map for semsis
		typeTable := internal.NewCache[string, types.TypeSpec]()
		for _, importStmt := range lib.Imports {
			importName := "/Users/personal/Documents/GitHub/dash/src/" + importStmt.Name.TokenLiteral()

			if importedLib, ok := libs[importName]; ok {
				extractPublicTypes(importedLib, typeTable)
			} else {
				panic("unable to find libary " + importName)
			}
		}

		s := semantic.New(dep.Path, typeTable)
		s.Analyse(lib)

		//  check for semantical errors
		if len(s.Errors()) > 0 {
			sErrs = append(sErrs, s.Errors()...)
		}

		if _, ok := libs[lib.Name.String()]; ok {
			return fmt.Errorf("duplicate library name '%s' found in directory '%s'",
				lib.Name.String(), dep.Path)
		}
		libs[dep.Path] = lib

		return nil
	}

	root, err := buildDependencyTree(cfg.SrcDir, filesPerDir)
	if err != nil {
		return nil, fmt.Errorf("unable to build dependency tree: %w", err)
	}

	walkDependencyTree(root, parse)

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

// ------- //
// Helpers //
// ------- //

func findProjectRoot(startDir string) (string, error) {
	// get absolute path
	absDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("unable to get absolute path for %s: %w", startDir, err)
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
			return "", fmt.Errorf("project root not found: no dash.toml file found in directory tree from %s", startDir)
		}
		currentDir = parentDir
	}
}

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
	Visitied bool
	Path     string
	// files that make up the library
	Files []string
	// libraries imported by current library
	Imports []*dependencyTree
}

func buildDependencyTree(entryLib string, filesPerDir map[string][]string) (*dependencyTree, error) {
	// maps 'dir' to a DependecyTree node
	nodes := make(map[string]*dependencyTree, len(filesPerDir))
	// contains all imports made by a library using 'dir' as key
	importMap := make(map[string][]string)
	for dir, files := range filesPerDir {
		nodes[dir] = &dependencyTree{Path: dir, Files: files}
		if strings.Contains(dir, entryLib) {
			entryLib = dir
		}
		for _, file := range files {
			// skip if already parsed
			if _, exists := importMap[file]; exists {
				continue
			}
			lib, err := extractImports(file)
			if err != nil {
				return nil, err
			}

			for _, imp := range lib.Imports {
				importMap[dir] = append(importMap[dir], "/Users/personal/Documents/GitHub/dash/src/"+imp.Name.TokenLiteral())
			}
		}
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

func walkDependencyTree(parent *dependencyTree, visit func(dep *dependencyTree) error) {
	for _, child := range parent.Imports {
		if !child.Visitied {
			walkDependencyTree(child, visit)
		}
	}

	parent.Visitied = true
	visit(parent)
}

func extractPublicTypes(lib *ast.Library, typeTable *internal.Cache[string, types.TypeSpec]) {
	// BUG: if we only use type name then its not visible we also need the lib name e.g. name.type
	// but then we also need to modify semsis
	for _, typeDef := range lib.TypeDefinitions {
		if typeDef.Public {
			typeTable.Set(typeDef.Name.String(), typeDef.T)
		}
	}

	for _, typeAlias := range lib.TypeAliases {
		if typeAlias.Public {
			typeTable.Set(typeAlias.Name.String(), typeAlias.T)
		}
	}

	for _, genericStruct := range lib.GenericStructs {
		if genericStruct.Public {
			typeTable.Set(genericStruct.Name.String(), genericStruct.T)
		}
	}

	for _, structDef := range lib.Structs {
		if structDef.Public {
			typeTable.Set(structDef.Name.String(), structDef.T)
		}
	}

	for _, enum := range lib.Enums {
		if enum.Public {
			typeTable.Set(enum.Name.String(), enum.T)
		}
	}

	for _, union := range lib.Unions {
		if union.Public {
			typeTable.Set(union.Name.String(), union.T)
		}
	}

	for _, err := range lib.Errors {
		if err.Public {
			errType := &types.Error{Name: err.Name.String()}
			typeTable.Set(err.Name.String(), errType)
		}
	}
}
