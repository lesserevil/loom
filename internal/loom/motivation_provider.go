package loom

import (
	"time"

	"github.com/jordanhubbard/loom/internal/motivation"
)

// LoomStateProvider implements motivation.StateProvider using Loom's internal state
type LoomStateProvider struct {
	loom *Loom
}

// NewLoomStateProvider creates a new state provider backed by Loom
func NewLoomStateProvider(l *Loom) *LoomStateProvider {
	return &LoomStateProvider{loom: l}
}

// GetCurrentTime returns the current time
func (p *LoomStateProvider) GetCurrentTime() time.Time {
	return time.Now()
}

// GetBeadsWithUpcomingDeadlines returns beads with deadlines within the specified days
func (p *LoomStateProvider) GetBeadsWithUpcomingDeadlines(withinDays int) ([]motivation.BeadDeadlineInfo, error) {
	return nil, nil
}

// GetOverdueBeads returns beads that are past their deadline
func (p *LoomStateProvider) GetOverdueBeads() ([]motivation.BeadDeadlineInfo, error) {
	return nil, nil
}

// GetBeadsByStatus returns bead IDs with the specified status
func (p *LoomStateProvider) GetBeadsByStatus(status string) ([]string, error) {
	if p.loom.beadsManager == nil {
		return nil, nil
	}
	beads, err := p.loom.beadsManager.ListBeads(map[string]interface{}{"status": status})
	if err != nil {
		return nil, err
	}
	var result []string
	for _, b := range beads {
		result = append(result, b.ID)
	}
	return result, nil
}

// GetMilestones returns milestones for a project
func (p *LoomStateProvider) GetMilestones(projectID string) ([]*motivation.Milestone, error) {
	return nil, nil
}

// GetUpcomingMilestones returns milestones within the specified days
func (p *LoomStateProvider) GetUpcomingMilestones(withinDays int) ([]*motivation.Milestone, error) {
	return nil, nil
}

// GetIdleAgents returns IDs of agents that are currently idle
func (p *LoomStateProvider) GetIdleAgents() ([]string, error) {
	if p.loom.agentManager == nil {
		return nil, nil
	}
	var result []string
	for _, ag := range p.loom.agentManager.ListAgents() {
		if ag != nil && ag.Status == "idle" {
			result = append(result, ag.ID)
		}
	}
	return result, nil
}

// GetAgentsByRole returns agent IDs with the specified role
func (p *LoomStateProvider) GetAgentsByRole(role string) ([]string, error) {
	if p.loom.agentManager == nil {
		return nil, nil
	}
	agents := p.loom.agentManager.ListAgents()
	var result []string
	for _, a := range agents {
		if a.Role == role {
			result = append(result, a.ID)
		}
	}
	return result, nil
}

// GetProjectIdle returns whether a project has been idle for the specified duration
func (p *LoomStateProvider) GetProjectIdle(projectID string, duration time.Duration) (bool, error) {
	return false, nil
}

// GetSystemIdle returns whether the entire system has been idle for the specified duration
func (p *LoomStateProvider) GetSystemIdle(duration time.Duration) (bool, error) {
	return false, nil
}

// GetCurrentSpending returns current spending for the specified period
func (p *LoomStateProvider) GetCurrentSpending(period string) (float64, error) {
	return 0, nil
}

// GetBudgetThreshold returns the budget threshold for a project
func (p *LoomStateProvider) GetBudgetThreshold(projectID string) (float64, error) {
	return 0, nil
}

// GetPendingDecisions returns IDs of pending decisions
func (p *LoomStateProvider) GetPendingDecisions() ([]string, error) {
	if p.loom.decisionManager == nil {
		return nil, nil
	}
	decisions, err := p.loom.decisionManager.GetPendingDecisions(nil)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, d := range decisions {
		result = append(result, d.ID)
	}
	return result, nil
}

// GetUnprocessedExternalEvents returns unprocessed external events of the specified type
func (p *LoomStateProvider) GetUnprocessedExternalEvents(eventType string) ([]motivation.ExternalEvent, error) {
	return nil, nil
}
