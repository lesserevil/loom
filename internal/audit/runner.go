package audit

import (
	"context"
	"log"
	"time"
)

type Runner struct {
	projectID       string
	projectPath     string
	intervalMinutes int
	activity        *SelfAuditActivity
	beadCreator     BeadCreator
	stopCh          chan struct{}
}

func NewRunner(projectID, projectPath string, intervalMinutes int, beadCreator BeadCreator) *Runner {
	if intervalMinutes == 0 {
		intervalMinutes = 30
	}
	return &Runner{
		projectID:       projectID,
		projectPath:     projectPath,
		intervalMinutes: intervalMinutes,
		activity:        NewSelfAuditActivity(projectPath),
		beadCreator:     beadCreator,
		stopCh:          make(chan struct{}),
	}
}

func (r *Runner) Start(ctx context.Context) {
	log.Printf("[SelfAudit] Starting for project %s, interval %dm", r.projectID, r.intervalMinutes)

	ticker := time.NewTicker(time.Duration(r.intervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.runAudit(ctx)
		}
	}
}

func (r *Runner) Stop() {
	close(r.stopCh)
}

func (r *Runner) runAudit(ctx context.Context) {
	log.Printf("[SelfAudit] Running audit for project %s", r.projectID)

	output, err := r.activity.RunSelfAudit(ctx, SelfAuditInput{
		ProjectID:   r.projectID,
		ProjectPath: r.projectPath,
	})

	if err != nil {
		log.Printf("[SelfAudit] Audit failed: %v", err)
		return
	}

	log.Printf("[SelfAudit] %s (findings: %d)", output.Result.Summary, len(output.Result.Findings))

	if len(output.Result.Findings) == 0 {
		return
	}

	if r.beadCreator == nil {
		log.Printf("[SelfAudit] %d issues found but no bead creator configured", len(output.Result.Findings))
		return
	}

	newIDs, err := r.activity.FileBeadsForFindings(ctx, r.beadCreator, output.Result.Findings, r.projectID)
	if err != nil {
		log.Printf("[SelfAudit] Failed to file beads: %v", err)
		return
	}

	if len(newIDs) > 0 {
		log.Printf("[SelfAudit] Filed %d new beads for project %s", len(newIDs), r.projectID)
	}
}
