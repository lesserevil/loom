package patterns

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// Optimizer generates optimization recommendations for usage patterns
type Optimizer struct {
	config *AnalysisConfig
}

// NewOptimizer creates a new optimizer
func NewOptimizer(config *AnalysisConfig) *Optimizer {
	if config == nil {
		config = DefaultAnalysisConfig()
	}
	return &Optimizer{
		config: config,
	}
}

// GenerateRecommendations creates optimization recommendations for patterns
func (o *Optimizer) GenerateRecommendations(patterns []*UsagePattern) []*Optimization {
	var optimizations []*Optimization

	for _, pattern := range patterns {
		// Rate limiting recommendations
		if o.config.EnableRateLimiting && pattern.RequestFrequency > o.config.RateLimitThreshold {
			opt := o.createRateLimitOptimization(pattern)
			if opt != nil {
				optimizations = append(optimizations, opt)
			}
		}

		// Provider substitution recommendations (enhanced version)
		if o.config.EnableSubstitutions && pattern.Type == "provider-model" {
			opt := o.createEnhancedSubstitutionOptimization(pattern)
			if opt != nil {
				optimizations = append(optimizations, opt)
			}
		}
	}

	// Sort by projected savings
	sort.Slice(optimizations, func(i, j int) bool {
		return optimizations[i].ProjectedSavingsUSD > optimizations[j].ProjectedSavingsUSD
	})

	return optimizations
}

// createRateLimitOptimization creates a rate limiting recommendation
func (o *Optimizer) createRateLimitOptimization(pattern *UsagePattern) *Optimization {
	// Calculate potential savings from reduced request volume
	excessRequests := pattern.RequestFrequency - o.config.RateLimitThreshold
	if excessRequests <= 0 {
		return nil
	}

	potentialSavings := excessRequests * pattern.AvgCost
	monthlySavings := potentialSavings * 30 // Extrapolate to monthly

	return &Optimization{
		ID:                  uuid.New().String(),
		Type:                "rate-limit",
		Pattern:             pattern,
		Recommendation:      fmt.Sprintf("Implement rate limiting for %s (%.0f req/day exceeds threshold of %.0f)", pattern.GroupKey, pattern.RequestFrequency, o.config.RateLimitThreshold),
		CurrentCost:         pattern.TotalCost,
		ProjectedCost:       pattern.TotalCost - potentialSavings,
		ProjectedSavingsUSD: potentialSavings,
		MonthlySavingsUSD:   monthlySavings,
		ImpactRating:        getImpactRating(monthlySavings),
		QualityImpact:       "minimal", // Rate limiting has minimal quality impact
		AutoApplicable:      false,     // Requires configuration
		Confidence:          0.8,
	}
}

// Helper functions

func getImpactRating(monthlySavings float64) string {
	switch {
	case monthlySavings >= 1000:
		return "high"
	case monthlySavings >= 100:
		return "medium"
	default:
		return "low"
	}
}
