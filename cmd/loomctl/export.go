package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

func newExportCommand() *cobra.Command {
	var (
		outputFile string
		format     string
		include    string
		exclude    string
		projectID  string
		since      string
		compress   bool
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export database to JSON file",
		Long: `Export the complete Loom database state to a JSON file.

This includes all tables: providers, projects, agents, workflows, activity, logs, etc.
The export can be used for backups, migrations, or disaster recovery.`,
		Example: `  # Export everything to a file
  loomctl export --output backup.json

  # Export with pretty-printed JSON
  loomctl export --format json-pretty --output backup.json

  # Export only core and workflow data
  loomctl export --include core,workflow --output partial.json

  # Export data for a specific project
  loomctl export --project loom-self --output project-backup.json

  # Export only data created/updated since a date
  loomctl export --since 2026-02-01T00:00:00Z --output incremental.json

  # Export to stdout (pipe to jq or file)
  loomctl export`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()

			// Build query parameters
			params := url.Values{}
			if format != "" {
				params.Set("format", format)
			}
			if include != "" {
				params.Set("include", include)
			}
			if exclude != "" {
				params.Set("exclude", exclude)
			}
			if projectID != "" {
				params.Set("project_id", projectID)
			}
			if since != "" {
				params.Set("since", since)
			}
			if compress {
				params.Set("compress", "true")
			}

			// Make request
			data, err := client.get("/api/v1/export", params)
			if err != nil {
				return err
			}

			// Write to file or stdout
			if outputFile != "" {
				if err := os.WriteFile(outputFile, data, 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Export saved to: %s\n", outputFile)
			} else {
				// Write to stdout (don't pretty-print, already handled by server)
				fmt.Println(string(data))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().StringVar(&format, "format", "json", "Output format: json, json-pretty")
	cmd.Flags().StringVar(&include, "include", "", "Include only these groups (comma-separated): core,workflow,activity,tracking,logging,analytics,config")
	cmd.Flags().StringVar(&exclude, "exclude", "", "Exclude these groups (comma-separated)")
	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Export only data for this project ID")
	cmd.Flags().StringVar(&since, "since", "", "Export only data created/updated since this timestamp (RFC3339 format)")
	cmd.Flags().BoolVar(&compress, "compress", false, "Compress the export (gzip)")

	return cmd
}

func newImportCommand() *cobra.Command {
	var (
		strategy     string
		dryRun       bool
		validateOnly bool
	)

	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Import database from JSON file",
		Long: `Import Loom database state from a JSON export file.

This will restore all data from a previous export. The import strategy determines
how conflicts are handled:

- merge (default): Update existing records, insert new ones (INSERT OR REPLACE)
- replace: Delete all existing data first, then import (CAUTION: destructive!)
- fail-on-conflict: Abort if any record already exists

Use --dry-run to preview changes without committing them.
Use --validate-only to check if the file can be imported without making changes.`,
		Example: `  # Import with default merge strategy
  loomctl import backup.json

  # Preview what would be imported
  loomctl import backup.json --dry-run

  # Validate export file without importing
  loomctl import backup.json --validate-only

  # Replace all existing data (CAUTION!)
  loomctl import backup.json --strategy replace

  # Fail if any conflicts exist
  loomctl import backup.json --strategy fail-on-conflict`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]

			// Read file
			data, err := os.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			client := newClient()

			// Build URL with query parameters
			params := url.Values{}
			if strategy != "" {
				params.Set("strategy", strategy)
			}
			if dryRun {
				params.Set("dry_run", "true")
			}
			if validateOnly {
				params.Set("validate_only", "true")
			}

			path := "/api/v1/import"
			if len(params) > 0 {
				path += "?" + params.Encode()
			}

			// Make POST request
			u := fmt.Sprintf("%s%s", client.BaseURL, path)
			req, err := http.NewRequest("POST", u, bytes.NewReader(data))
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.HTTP.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			if resp.StatusCode >= 400 {
				return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(respBody))
			}

			// Output result
			outputJSON(respBody)

			if dryRun {
				fmt.Fprintf(os.Stderr, "\nDry run completed - no changes were made.\n")
			} else if validateOnly {
				fmt.Fprintf(os.Stderr, "\nValidation completed.\n")
			} else {
				fmt.Fprintf(os.Stderr, "\nImport completed successfully.\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&strategy, "strategy", "merge", "Import strategy: merge, replace, fail-on-conflict")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without committing")
	cmd.Flags().BoolVar(&validateOnly, "validate-only", false, "Validate file without importing")

	return cmd
}
