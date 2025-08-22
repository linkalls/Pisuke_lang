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
	"strings"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "build" {
		fmt.Println("Usage: pisuke build <filename>")
		os.Exit(1)
	}

	inputFile := os.Args[2]
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()

	// In a real compiler, we'd check for parser errors here.

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
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error compiling generated Go code: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully compiled %s to %s\n", inputFile, outputName)
}
