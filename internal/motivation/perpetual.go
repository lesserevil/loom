package motivation

import (
	"time"
)

// PerpetualTaskMotivations returns motivations for scheduled perpetual tasks
// These tasks run on regular intervals regardless of system activity
func PerpetualTaskMotivations() []*Motivation {
	return []*Motivation{
		// ============================================
		// CFO Perpetual Tasks
		// ============================================
		{
			Name:                "Daily Budget Review",
			Description:         "CFO reviews daily spending and budget utilization every 24 hours",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "cfo",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "daily-budget-review",
			Priority:            70,
			CooldownPeriod:      22 * time.Hour, // Slightly less than 24h to avoid drift
			Parameters: map[string]interface{}{
				"interval":  "24h",
				"task_type": "perpetual",
			},
			IsBuiltIn: true,
		},
		{
			Name:                "Weekly Cost Optimization Report",
			Description:         "CFO analyzes cost trends and identifies optimization opportunities weekly",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "cfo",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "weekly-cost-report",
			Priority:            65,
			CooldownPeriod:      7 * 24 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":  "168h", // 7 days
				"task_type": "perpetual",
			},
			IsBuiltIn: true,
		},

		// ============================================
		// QA Engineer Perpetual Tasks
		// ============================================
		{
			Name:                "Daily Automated Test Suite Run",
			Description:         "QA Engineer runs full automated test suite daily to ensure quality",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "qa-engineer",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "daily-test-run",
			Priority:            75,
			CooldownPeriod:      22 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":   "24h",
				"task_type":  "perpetual",
				"test_suite": "full",
			},
			IsBuiltIn: true,
		},
		{
			Name:                "Weekly Integration Test Review",
			Description:         "QA Engineer performs comprehensive integration testing weekly",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "qa-engineer",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "weekly-integration-tests",
			Priority:            70,
			CooldownPeriod:      7 * 24 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":  "168h",
				"task_type": "perpetual",
				"test_type": "integration",
			},
			IsBuiltIn: true,
		},
		{
			Name:                "Weekly Regression Test Sweep",
			Description:         "QA Engineer runs regression tests weekly to catch regressions early",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "qa-engineer",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "weekly-regression-tests",
			Priority:            72,
			CooldownPeriod:      7 * 24 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":  "168h",
				"task_type": "perpetual",
				"test_type": "regression",
			},
			IsBuiltIn: true,
		},

		// ============================================
// PR Manager Perpetual Tasks
		// ============================================
		{
			Name:                "CI/CD Pipeline Monitoring",
			Description:         "PR Manager checks CI/CD pipeline status and files devops beads on failure",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "public-relations-manager",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "cicd-pipeline-monitoring",
			Priority:            65,
			CooldownPeriod:      1 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":  "1h",
				"task_type": "perpetual",
			},
			IsBuiltIn: true,
			Execute: func(ctx context.Context, client *github.Client) error {
				// Import necessary packages
				import (
					"context"
					"fmt"
					"strings"
					"github.com/google/go-github/v33/github"
				)

				// Define Bead struct
				type Bead struct {
					Type        string
					Priority    string
					Title       string
					Description string
					AssignedRole string
				}

				// Implement the logic

				failedRuns, err := client.ListFailedWorkflowRuns(ctx)
				if err != nil {
					return fmt.Errorf("failed to list workflow runs: %w", err)
				}

				for _, run := range failedRuns {
					// Check for existing open bead with the same title prefix
					// If not found, file a new bead
					// Implement deduplication logic here
					// Check for existing open bead with the same title prefix
					openBeads, err := client.ListOpenBeads(ctx, "CI/CD pipeline failing")
					if err != nil {
						return fmt.Errorf("failed to list open beads: %w", err)
					}

					for _, bead := range openBeads {
						if strings.HasPrefix(bead.Title, "CI/CD pipeline failing: "+run.Name) {
							// Bead already exists, skip filing
							continue
						}
					}

					// File a new bead
					bead := &Bead{
						Type:        "task",
						Priority:    "P1",
						Title:       "CI/CD pipeline failing: " + run.Name,
						Description: fmt.Sprintf("Failed workflow URL: %s", run.URL),
						AssignedRole: "devops-engineer",
					}
					if err := client.FileBead(ctx, bead); err != nil {
						return fmt.Errorf("failed to file bead: %w", err)
					}

				}

				return nil
			},
		}, 
			IsBuiltIn: true,
		},
		{
			Name:                "CI/CD Pipeline Monitoring",
			Description:         "PR Manager checks CI/CD pipeline status and files devops beads on failure",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "public-relations-manager",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "cicd-pipeline-monitoring",
			Priority:            65,
			CooldownPeriod:      1 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":  "1h",
				"task_type": "perpetual",
			},
			IsBuiltIn: true,
		},
		{
			Name:                "Daily Community Engagement Report",
			Description:         "PR Manager reviews and reports on community engagement metrics daily",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "public-relations-manager",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "daily-community-report",
			Priority:            55,
			CooldownPeriod:      22 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":  "24h",
				"task_type": "perpetual",
			},
			IsBuiltIn: true,
		},

		// ============================================
		// Documentation Manager Perpetual Tasks
		// ============================================
		{
			Name:                "Daily Documentation Audit",
			Description:         "Documentation Manager reviews and updates documentation daily",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "documentation-manager",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "daily-docs-audit",
			Priority:            50,
			CooldownPeriod:      22 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":   "24h",
				"task_type":  "perpetual",
				"audit_type": "automated",
			},
			IsBuiltIn: true,
		},
		{
			Name:                "Weekly Documentation Consistency Check",
			Description:         "Documentation Manager ensures documentation consistency across the project weekly",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "documentation-manager",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "weekly-docs-consistency",
			Priority:            55,
			CooldownPeriod:      7 * 24 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":  "168h",
				"task_type": "perpetual",
			},
			IsBuiltIn: true,
		},

		// ============================================
		// DevOps Engineer Perpetual Tasks
		// ============================================
		{
			Name:                "Daily Infrastructure Health Check",
			Description:         "DevOps Engineer performs daily infrastructure health and monitoring review",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "devops-engineer",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "daily-infra-health",
			Priority:            75,
			CooldownPeriod:      22 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":  "24h",
				"task_type": "perpetual",
			},
			IsBuiltIn: true,
		},
		{
			Name:                "Weekly Security Audit",
			Description:         "DevOps Engineer performs weekly security audit and vulnerability scanning",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "devops-engineer",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "weekly-security-audit",
			Priority:            80,
			CooldownPeriod:      7 * 24 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":   "168h",
				"task_type":  "perpetual",
				"audit_type": "security",
			},
			IsBuiltIn: true,
		},

		// ============================================
		// Project Manager Perpetual Tasks
		// ============================================
		{
			Name:                "Daily Standup Review",
			Description:         "Project Manager reviews daily progress, blockers, and team status",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "project-manager",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "daily-standup",
			Priority:            70,
			CooldownPeriod:      22 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":  "24h",
				"task_type": "perpetual",
			},
			IsBuiltIn: true,
		},
		{
			Name:                "Weekly Sprint Planning",
			Description:         "Project Manager conducts weekly sprint planning and retrospective",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "project-manager",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "weekly-sprint-planning",
			Priority:            75,
			CooldownPeriod:      7 * 24 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":  "168h",
				"task_type": "perpetual",
			},
			IsBuiltIn: true,
		},

		// ============================================
		// Housekeeping Bot Perpetual Tasks
		// ============================================
		{
			Name:                "Hourly Cleanup Tasks",
			Description:         "Housekeeping Bot performs routine cleanup tasks every hour",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "housekeeping-bot",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "hourly-cleanup",
			Priority:            30,
			CooldownPeriod:      55 * time.Minute,
			Parameters: map[string]interface{}{
				"interval":  "1h",
				"task_type": "perpetual",
			},
			IsBuiltIn: true,
		},
		{
			Name:                "Weekly Data Archival",
			Description:         "Housekeeping Bot archives old data and logs weekly",
			Type:                MotivationTypeCalendar,
			Condition:           ConditionScheduledInterval,
			AgentRole:           "housekeeping-bot",
			WakeAgent:           true,
			CreateBeadOnTrigger: true,
			BeadTemplate:        "weekly-archival",
			Priority:            35,
			CooldownPeriod:      7 * 24 * time.Hour,
			Parameters: map[string]interface{}{
				"interval":  "168h",
				"task_type": "perpetual",
			},
			IsBuiltIn: true,
		},
	}
}

// RegisterPerpetualTasks registers all perpetual task motivations with the registry
func RegisterPerpetualTasks(registry *Registry) error {
	perpetual := PerpetualTaskMotivations()
	for _, m := range perpetual {
		if err := registry.Register(m); err != nil {
			// Skip duplicates silently
			continue
		}
	}
	return nil
}

// GetPerpetualTasksByRole returns perpetual task motivations for a specific role
func GetPerpetualTasksByRole(role string) []*Motivation {
	perpetual := PerpetualTaskMotivations()
	result := make([]*Motivation, 0)
	for _, m := range perpetual {
		if m.AgentRole == role {
			result = append(result, m)
		}
	}
	return result
}
