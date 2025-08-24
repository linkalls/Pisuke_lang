package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"pisuke/codegen"
	"pisuke/lexer"
	"pisuke/parser"
	"pisuke/token"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: pisuke <command> <filename>")
		fmt.Println("Commands: build, debug")
		os.Exit(1)
	}

	command := os.Args[1]
	inputFile := os.Args[2]
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		os.Exit(1)
	}

	// Preprocess imports: inline referenced .psk modules and remove import statements
	processed, err := preprocessImports(inputFile, string(data))
	if err != nil {
		fmt.Printf("Error processing imports: %s\n", err)
		os.Exit(1)
	}

	l := lexer.New(processed)

	switch command {
	case "debug":
		fmt.Println("--- Tokens ---")
		for {
			tok := l.NextToken()
			fmt.Printf("%+v\n", tok)
			if tok.Type == token.EOF {
				break
			}
		}
		// Re-create lexer because it's stateful; use processed content (imports inlined)
		l = lexer.New(processed)
		p := parser.New(l)
		program := p.ParseProgram()
		fmt.Println("\n--- AST ---")
		fmt.Println(program.String())

		fmt.Println("\n--- Generated Go Code ---")
		generatedCode := codegen.Generate(program)
		fmt.Println(generatedCode)

	case "build":
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors) > 0 {
			fmt.Println("Parser errors:")
			for _, msg := range p.Errors {
				fmt.Println("\t" + msg)
			}
			os.Exit(1)
		}

		generatedCode := codegen.Generate(program)
		tempGoFile := "pisuke_temp_output.go"
		err = ioutil.WriteFile(tempGoFile, []byte(generatedCode), 0644)
		if err != nil {
			fmt.Printf("Error writing temporary Go file: %s\n", err)
			os.Exit(1)
		}
		defer os.Remove(tempGoFile)

		outputName := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))

		cmd := exec.Command("go", "build", "-o", outputName, tempGoFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()

		if err != nil {
			fmt.Printf("Error compiling generated Go code: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully compiled %s to %s\n", inputFile, outputName)
	}
}

// preprocessImports finds import statements like: import { a, b } from "module"
// and replaces them by inlining the contents of the referenced .psk file(s).
// It resolves relative paths based on the importing file's directory. It avoids
// duplicating the same module by tracking visited files.
func preprocessImports(entryFile string, content string) (string, error) {
	visited := make(map[string]bool)
	dir := filepath.Dir(entryFile)
	return resolveImportsRecursive(dir, content, visited)
}

// resolveImportsRecursive scans content for import statements, loads referenced
// files and inlines them. It returns the resulting source where import lines
// are removed and replaced by the inlined module source.
func resolveImportsRecursive(baseDir string, content string, visited map[string]bool) (string, error) {
	// regex to match: import { ... } from "module"
	re := regexp.MustCompile(`import\s*\{[^}]*\}\s*from\s*"([^"]+)"`)

	result := content
	matches := re.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		modulePath := m[1]
		// resolve to a .psk file path. Support modulePath like "math" or "std/webserver"
		var candidate string
		if filepath.IsAbs(modulePath) {
			candidate = modulePath + ".psk"
		} else {
			candidate = filepath.Join(baseDir, modulePath+".psk")
		}

		// If the file doesn't exist relative, try modulePath directly in workspace
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			// try modulePath as-is (maybe already contains path separators)
			candidate = modulePath
			if !strings.HasSuffix(candidate, ".psk") {
				candidate = candidate + ".psk"
			}
		}

		abs, err := filepath.Abs(candidate)
		if err != nil {
			return "", err
		}
		if visited[abs] {
			// already inlined; simply remove the import
			result = strings.Replace(result, m[0], "", -1)
			continue
		}

		data, err := ioutil.ReadFile(abs)
		if err != nil {
			return "", fmt.Errorf("cannot read module %s: %w", modulePath, err)
		}
		visited[abs] = true

		// Recursively resolve imports in the module itself
		inlined, err := resolveImportsRecursive(filepath.Dir(abs), string(data), visited)
		if err != nil {
			return "", err
		}

		// Replace the import statement with the inlined module source
		result = strings.Replace(result, m[0], "\n// begin inlined module: "+modulePath+"\n"+inlined+"\n// end inlined module: "+modulePath+"\n", -1)
	}
	return result, nil
}
