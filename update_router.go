package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	content, err := os.ReadFile("internal/actions/router.go")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	text := string(content)

	// Add persona import after files import
	if !strings.Contains(text, "internal/persona") {
		text = strings.Replace(text,
			`"github.com/jordanhubbard/loom/internal/files"`,
			`"github.com/jordanhubbard/loom/internal/files"
	"github.com/jordanhubbard/loom/internal/persona"`,
			1)
		fmt.Println("Added persona import")
	}

	// Add PersonaManager field after Voter field
	if !strings.Contains(text, "PersonaManager") {
		text = strings.Replace(text,
			`	Voter         VoteCaster
	BeadType      string`,
			`	Voter         VoteCaster
	PersonaManager *persona.Manager
	BeadType      string`,
			1)
		fmt.Println("Added PersonaManager field")
	}

	err = os.WriteFile("internal/actions/router.go", []byte(text), 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}

	fmt.Println("Successfully updated router.go")
}
