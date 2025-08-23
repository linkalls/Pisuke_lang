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

	l := lexer.New(string(data))

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
		// Re-create lexer because it's stateful
		l = lexer.New(string(data))
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
