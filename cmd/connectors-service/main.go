// Package main implements the connectors service for Loom.
package main

import (
	"github.com/jordanhubbard/loom/internal/github"
)

func main() {
	// Start the GitHub webhook server
	github.StartWebhookServer()
}
