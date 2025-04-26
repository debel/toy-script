package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	switch len(os.Args) {
	case 1:
		err := runPrompt()
		if err != nil {
			log.Fatalln(err)
		}
	case 3:
		cmd := os.Args[1]
		scriptPath := os.Args[2]
		switch cmd {
		case "run":
			err := runScript(scriptPath)
			if err != nil {
				log.Fatalln(err)
			}
		case "build":
			log.Fatalln("Build command not implemented yet")
		default:
			log.Fatalf("Unknown command %s.\nSupported commands are: run | build\n", cmd)
		}
	default:
		log.Fatalln("Usage: toyscript [command] [script]")
	}
}

func runScript(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	contents, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	script := string(contents)

	run(script)
	return nil
}

func runPrompt() error {
	fmt.Println("debel-toy-lang v0.0.1")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		cmd, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		cmd = strings.TrimSpace(cmd)
		switch cmd {
		case "exit", "":
			os.Exit(0)
		default:
			run(cmd)
		}
	}
}

func run(source string) error {
	scanner := NewScanner(source)

	tokens, err := scanner.ScanTokens()
	if err != nil {
		return err
	}

	parser := NewParser(tokens)

	ast, hasErr := parser.Parse()
	if hasErr {
		fmt.Println("parsing error: ", ast.String())
		return err
	}

	// fmt.Println("---- ast ----")
	// fmt.Println(ast.String())
	// fmt.Println("---- ast ----")
	//
	// fmt.Println("---- eval ----")
	evaler := NewInterpreter(map[string]inode{})
	evaler.Exec(&ast)
	// fmt.Println("---- eval ----")

	return nil
}
