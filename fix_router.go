package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	content, _ := os.ReadFile("internal/actions/router.go")
	text := string(content)

	// Replace the import section
	old := `	"github.com/jordanhubbard/loom/internal/files"
	"github.com/jordanhubbard/loom/pkg/models"`
	new := `	"github.com/jordanhubbard/loom/internal/files"
	"github.com/jordanhubbard/loom/internal/persona"
	"github.com/jordanhubbard/loom/pkg/models"`

	if strings.Contains(text, old) {
		text = strings.Replace(text, old, new, 1)
		fmt.Println("Updated imports")
	}

	// Replace Router struct
	old2 := `	Voter         VoteCaster
	BeadType      string`
	new2 := `	Voter         VoteCaster
	PersonaManager *persona.Manager
	BeadType      string`

	if strings.Contains(text, old2) {
		text = strings.Replace(text, old2, new2, 1)
		fmt.Println("Updated Router struct")
	}

	os.WriteFile("internal/actions/router.go", []byte(text), 0644)
	fmt.Println("Done")
}
