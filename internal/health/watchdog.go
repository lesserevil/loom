package health

import (
	"context"
	"log"
	"time"

	"github.com/jordanhubbard/loom/internal/beads"
	"github.com/jordanhubbard/loom/internal/metrics"
	"github.com/jordanhubbard/loom/internal/provider"
)

// Watchdog periodically checks the system health and creates alerts if needed.
type Watchdog struct {
	beadsMgr   *beads.Manager
	metricsMgr *metrics.Metrics
	providerReg *provider.Registry
}

// NewWatchdog creates a new Watchdog instance.
func NewWatchdog(beadsMgr *beads.Manager, metricsMgr *metrics.Metrics, providerReg *provider.Registry) *Watchdog {
	return &Watchdog{
		beadsMgr:   beadsMgr,
		metricsMgr: metricsMgr,
		providerReg: providerReg,
	}
}

// Start begins the watchdog process.
func (w *Watchdog) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.checkHealth(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// checkHealth performs the health checks and creates alerts if necessary.
func (w *Watchdog) checkHealth(ctx context.Context) {
	log.Println("[Watchdog] Performing health check")

	// Check for projects with 0 in_progress beads and N+ open beads for >30 minutes
	// Check context-canceled error rate
	// Check for zombie beads
	// Check if Ralph is blocking >50% of a project's beads

	// Implement health checks
	projects, err := w.beadsMgr.ListProjects()
	if err != nil {
		log.Printf("[Watchdog] Error listing projects: %v", err)
		return
	}

	for _, project := range projects {
		openBeads, err := w.beadsMgr.ListBeads(map[string]interface{}{"status": "open", "project_id": project.ID})
		if err != nil {
			log.Printf("[Watchdog] Error listing open beads for project %s: %v", project.ID, err)
			continue
		}

		inProgressBeads, err := w.beadsMgr.ListBeads(map[string]interface{}{"status": "in_progress", "project_id": project.ID})
		if err != nil {
			log.Printf("[Watchdog] Error listing in-progress beads for project %s: %v", project.ID, err)
			continue
		}

		// Check for projects with 0 in_progress beads and N+ open beads for >30 minutes
		if len(inProgressBeads) == 0 && len(openBeads) > 5 {
			log.Printf("[Watchdog] Project %s has 0 in-progress beads and %d open beads", project.ID, len(openBeads))
			// Create a P0 bead assigned to the CEO
			w.createAlertBead(project.ID, "No progress on open beads")
		}
	}

	// Placeholder for additional health checks
}

// createAlertBead creates a P0 bead assigned to the CEO
func (w *Watchdog) createAlertBead(projectID, reason string) {
	log.Printf("[Watchdog] Creating alert bead for project %s: %s", projectID, reason)
	// Placeholder for bead creation logic
}
}
