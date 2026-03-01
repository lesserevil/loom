package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, err := os.ReadFile("internal/actions/router.go")
	if err != nil {
		fmt.Printf("read error: %v\n", err)
		return
	}

	content := string(b)

	// Step 1: Add import
	if !strings.Contains(content, "internal/persona") {
		content = strings.Replace(
			content,
			`"github.com/jordanhubbard/loom/internal/files"
	"github.com/jordanhubbard/loom/pkg/models"`,
			`"github.com/jordanhubbard/loom/internal/files"
	"github.com/jordanhubbard/loom/internal/persona"
	"github.com/jordanhubbard/loom/pkg/models"`,
			1,
		)
		fmt.Println("Added persona import")
	}

	// Step 2: Add PersonaManager field
	if !strings.Contains(content, "PersonaManager") {
		content = strings.Replace(
			content,
			`	Voter         VoteCaster
	BeadType      string`,
			`	Voter         VoteCaster
	PersonaManager *persona.Manager
	BeadType      string`,
			1,
		)
		fmt.Println("Added PersonaManager field")
	}

	err = os.WriteFile("internal/actions/router.go", []byte(content), 0644)
	if err != nil {
		fmt.Printf("write error: %v\n", err)
		return
	}

	fmt.Println("File updated successfully")
}
