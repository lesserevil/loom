package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type docEntry struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Section string `json:"section"`
}

type docContent struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// handleDocs serves the documentation index and individual pages.
// GET /api/v1/docs         -> list all doc pages
// GET /api/v1/docs?path=X  -> get content of a specific page
func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	docsRoot := "./docs"

	path := r.URL.Query().Get("path")
	if path != "" {
		s.serveDocPage(w, docsRoot, path)
		return
	}

	s.serveDocIndex(w, docsRoot)
}

func (s *Server) serveDocIndex(w http.ResponseWriter, docsRoot string) {
	entries := make([]docEntry, 0)

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		relPath, _ := filepath.Rel(docsRoot, path)

		if strings.HasPrefix(relPath, "archive/") {
			return nil
		}

		title := extractTitle(path)
		section := classifySection(relPath)

		entries = append(entries, docEntry{
			Path:    relPath,
			Title:   title,
			Section: section,
		})
		return nil
	}

	filepath.Walk(docsRoot, walkFn)

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Section != entries[j].Section {
			return entries[i].Section < entries[j].Section
		}
		return entries[i].Title < entries[j].Title
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"docs":  entries,
		"count": len(entries),
	})
}

func (s *Server) serveDocPage(w http.ResponseWriter, docsRoot, reqPath string) {
	clean := filepath.Clean(reqPath)
	if strings.Contains(clean, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(docsRoot, clean)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	title := extractTitleFromContent(string(data))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docContent{
		Path:    clean,
		Title:   title,
		Content: string(data),
	})
}

func extractTitle(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return filepath.Base(path)
	}
	return extractTitleFromContent(string(data))
}

func extractTitleFromContent(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "Untitled"
}

func classifySection(relPath string) string {
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) >= 2 {
		switch parts[0] {
		case "getting-started":
			return "Getting Started"
		case "guide":
			if len(parts) >= 3 {
				switch parts[1] {
				case "user":
					return "User Guide"
				case "admin":
					return "Administrator Guide"
				case "developer":
					return "Developer Guide"
				case "reference":
					return "Reference"
				case "tutorials":
					return "Tutorials"
				}
			}
		}
	}
	return "General"
}
