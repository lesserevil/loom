package main

import (
	"os"
	"strings"
)

func main() {
	data, err := os.ReadFile("web/static/index.html")
	if err != nil {
		panic(err)
	}
	content := string(data)
	
	// Fix duplicate emojis in tabs
	// Change Logs from 📋 to 📜 (scroll)
	content = strings.Replace(content, 
		`<button class="view-tab" data-target="logs" role="tab" aria-selected="false">📋 Logs</button>`,
		`<button class="view-tab" data-target="logs" role="tab" aria-selected="false">📜 Logs</button>`,
		1)
	
	// Change Diagrams from 📊 to 🔀 (flow)
	content = strings.Replace(content,
		`<button class="view-tab" data-target="diagrams" role="tab" aria-selected="false">📊 Diagrams</button>`,
		`<button class="view-tab" data-target="diagrams" role="tab" aria-selected="false">🔀 Diagrams</button>`,
		1)
	
	// Also fix the Logs section header to match
	content = strings.Replace(content,
		`<h2><span aria-hidden="true">📋</span> System Logs & Metrics</h2>`,
		`<h2><span aria-hidden="true">📜</span> System Logs & Metrics</h2>`,
		1)
	
	// Remove legacy REPL section
	content = strings.Replace(content,
		"        <!-- Legacy REPL section - kept for backward compatibility but hidden by default -->\n        <section id=\"repl\" class=\"view-panel\" role=\"tabpanel\" aria-label=\"CEO REPL (Legacy)\" hidden style=\"display: none;\">\n            <!-- Content moved to unified CEO section -->\n        </section>\n",
		"",
		1)
	
	// Remove legacy streaming-test section
	content = strings.Replace(content,
		"        <!-- Legacy Streaming Test section - kept for backward compatibility -->\n        <section id=\"streaming-test\" class=\"view-panel\" role=\"tabpanel\" aria-label=\"Streaming Test (Legacy)\" hidden style=\"display: none;\">\n            <!-- Content moved to dev-tools section -->\n        </section>\n",
		"",
		1)
	
	// Remove legacy project-list div
	content = strings.Replace(content,
		"            <div id=\"project-list\" class=\"project-container\" style=\"display: none;\">\n                <!-- Legacy project list (kept for backward compatibility) -->\n            </div>\n",
		"",
		1)
	
	err = os.WriteFile("web/static/index.html", []byte(content), 0644)
	if err != nil {
		panic(err)
	}
	println("Fixed UI duplications")
}
