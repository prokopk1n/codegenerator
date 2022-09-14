package main

import (
	"go/parser"
	"go/token"
	"log"
	"os"
)

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	out, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatalf("Can not create file: %v", err)
	}

	serveHTTPmap, err := GoThroughDecls(node)
	if err != nil {
		log.Fatalf("Can not generate code: %v", err)
	}

	err = GenerateCode(out, serveHTTPmap, node.Name.Name)
	if err != nil {
		log.Fatalf("Error in generate code: %v", err)
	}
}
