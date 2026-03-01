package feedback

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jordanhubbard/loom/internal/database"
	"github.com/jordanhubbard/loom/internal/eventbus"
)

// Manager handles feedback operations
type Manager struct {
	db       *database.Database
	eventBus *eventbus.EventBus
}

// Feedback represents user feedback on a bead or agent action
type Feedback struct {
	ID        string                 `json:"id"`
	BeadID    string                 `json:"bead_id,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	AuthorID  string                 `json:"author_id"`
	Author    string                 `json:"author"`
	Rating    int                    `json:"rating"` // 1-5 scale
	Category  string                 `json:"category"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// NewManager creates a new feedback manager
func NewManager(db *database.Database, eventBus *eventbus.EventBus) *Manager {
	return &Manager{
		db:       db,
		eventBus: eventBus,
	}
}

// CreateFeedback creates new feedback
func (m *Manager) CreateFeedback(beadID, agentID, authorID, author, category, content string, rating int, metadata map[string]interface{}) (*Feedback, error) {
	if rating < 1 || rating > 5 {
		return nil, fmt.Errorf("rating must be between 1 and 5")
	}

	if category == "" {
		return nil, fmt.Errorf("category is required")
	}

	now := time.Now()
	feedback := &Feedback{
		ID:        uuid.New().String(),
		BeadID:    beadID,
		AgentID:   agentID,
		AuthorID:  authorID,
		Author:    author,
		Rating:    rating,
		Category:  category,
		Content:   content,
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save to database
	dbFeedback := &database.Feedback{
		ID:        feedback.ID,
		BeadID:    feedback.BeadID,
		AgentID:   feedback.AgentID,
		AuthorID:  feedback.AuthorID,
		Author:    feedback.Author,
		Rating:    feedback.Rating,
		Category:  feedback.Category,
		Content:   feedback.Content,
		Metadata:  feedback.Metadata,
		CreatedAt: feedback.CreatedAt,
		UpdatedAt: feedback.UpdatedAt,
	}

	if err := m.db.CreateFeedback(dbFeedback); err != nil {
		return nil, err
	}

	// Publish event to EventBus
	if m.eventBus != nil {
		m.publishFeedbackEvent("feedback.created", feedback)
	}

	return feedback, nil
}

// GetFeedback retrieves feedback by ID
func (m *Manager) GetFeedback(feedbackID string) (*Feedback, error) {
	dbFeedback, err := m.db.GetFeedback(feedbackID)
	if err != nil {
		return nil, err
	}

	return &Feedback{
		ID:        dbFeedback.ID,
		BeadID:    dbFeedback.BeadID,
		AgentID:   dbFeedback.AgentID,
		AuthorID:  dbFeedback.AuthorID,
		Author:    dbFeedback.Author,
		Rating:    dbFeedback.Rating,
		Category:  dbFeedback.Category,
		Content:   dbFeedback.Content,
		Metadata:  dbFeedback.Metadata,
		CreatedAt: dbFeedback.CreatedAt,
		UpdatedAt: dbFeedback.UpdatedAt,
	}, nil
}

// GetFeedbackByBead retrieves all feedback for a bead
func (m *Manager) GetFeedbackByBead(beadID string) ([]*Feedback, error) {
	dbFeedbacks, err := m.db.GetFeedbackByBeadID(beadID)
	if err != nil {
		return nil, err
	}

	var feedbacks []*Feedback
	for _, dbFeedback := range dbFeedbacks {
		feedbacks = append(feedbacks, &Feedback{
			ID:        dbFeedback.ID,
			BeadID:    dbFeedback.BeadID,
			AgentID:   dbFeedback.AgentID,
			AuthorID:  dbFeedback.AuthorID,
			Author:    dbFeedback.Author,
			Rating:    dbFeedback.Rating,
			Category:  dbFeedback.Category,
			Content:   dbFeedback.Content,
			Metadata:  dbFeedback.Metadata,
			CreatedAt: dbFeedback.CreatedAt,
			UpdatedAt: dbFeedback.UpdatedAt,
		})
	}

	return feedbacks, nil
}

// GetFeedbackByAgent retrieves all feedback for an agent
func (m *Manager) GetFeedbackByAgent(agentID string) ([]*Feedback, error) {
	dbFeedbacks, err := m.db.GetFeedbackByAgentID(agentID)
	if err != nil {
		return nil, err
	}

	var feedbacks []*Feedback
	for _, dbFeedback := range dbFeedbacks {
		feedbacks = append(feedbacks, &Feedback{
			ID:        dbFeedback.ID,
			BeadID:    dbFeedback.BeadID,
			AgentID:   dbFeedback.AgentID,
			AuthorID:  dbFeedback.AuthorID,
			Author:    dbFeedback.Author,
			Rating:    dbFeedback.Rating,
			Category:  dbFeedback.Category,
			Content:   dbFeedback.Content,
			Metadata:  dbFeedback.Metadata,
			CreatedAt: dbFeedback.CreatedAt,
			UpdatedAt: dbFeedback.UpdatedAt,
		})
	}

	return feedbacks, nil
}

// UpdateFeedback updates feedback
func (m *Manager) UpdateFeedback(feedbackID, authorID, category, content string, rating int) error {
	// Verify ownership
	dbFeedback, err := m.db.GetFeedback(feedbackID)
	if err != nil {
		return err
	}

	if dbFeedback.AuthorID != authorID {
		return fmt.Errorf("unauthorized: only the author can edit their feedback")
	}

	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	// Update feedback
	if err := m.db.UpdateFeedback(feedbackID, category, content, rating); err != nil {
		return err
	}

	// Publish event
	if m.eventBus != nil {
		feedback := &Feedback{
			ID:        feedbackID,
			BeadID:    dbFeedback.BeadID,
			AgentID:   dbFeedback.AgentID,
			AuthorID:  authorID,
			Rating:    rating,
			Category:  category,
			Content:   content,
			UpdatedAt: time.Now(),
		}
		m.publishFeedbackEvent("feedback.updated", feedback)
	}

	return nil
}

// DeleteFeedback deletes feedback
func (m *Manager) DeleteFeedback(feedbackID, authorID string) error {
	// Verify ownership
	dbFeedback, err := m.db.GetFeedback(feedbackID)
	if err != nil {
		return err
	}

	if dbFeedback.AuthorID != authorID {
		return fmt.Errorf("unauthorized: only the author can delete their feedback")
	}

	// Delete feedback
	if err := m.db.DeleteFeedback(feedbackID); err != nil {
		return err
	}

	// Publish event
	if m.eventBus != nil {
		feedback := &Feedback{
			ID:       feedbackID,
			BeadID:   dbFeedback.BeadID,
			AgentID:  dbFeedback.AgentID,
			AuthorID: authorID,
		}
		m.publishFeedbackEvent("feedback.deleted", feedback)
	}

	return nil
}

// GetFeedbackStats returns statistics for feedback
func (m *Manager) GetFeedbackStats(beadID, agentID string) (map[string]interface{}, error) {
	stats, err := m.db.GetFeedbackStats(beadID, agentID)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// publishFeedbackEvent publishes a feedback event to the EventBus
func (m *Manager) publishFeedbackEvent(eventType string, feedback *Feedback) {
	event := &eventbus.Event{
		ID:        uuid.New().String(),
		Type:      eventbus.EventType(eventType),
		Timestamp: time.Now(),
		Source:    "feedback",
		Data: map[string]interface{}{
			"feedback_id": feedback.ID,
			"bead_id":     feedback.BeadID,
			"agent_id":    feedback.AgentID,
			"author_id":   feedback.AuthorID,
			"author":      feedback.Author,
			"rating":      feedback.Rating,
			"category":    feedback.Category,
			"content":     feedback.Content,
		},
	}

	_ = m.eventBus.Publish(event)
}
