// Package router contains internal routing contracts shared between the
// classifier, policy engine, route decision engine, and request processors.
package router

type JobDescriptor struct {
	RequestID               string
	TenantID                string
	ProjectID               string
	TaskType                string
	RiskLevel               string
	Sensitivity             string
	PromptTokensEstimate    int
	MaxOutputTokensEstimate int
	RequiresReasoning       bool
	RequiresCode            bool
	RequiresToolUse         bool
	RequiresJSONSchema      bool
	RequiresLargeContext    bool
	RequiresVision          bool
	LatencyPreference       string
	QualityPreference       string
	BudgetPreference        string
	FilesTouched            []string
	Keywords                []string
	RouterMode              string
	ExplicitModel           *string
	Metadata                map[string]any
}
