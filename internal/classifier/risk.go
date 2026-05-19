package classifier

const (
	RiskLow      = "low"
	RiskMedium   = "medium"
	RiskHigh     = "high"
	RiskCritical = "critical"

	SensitivityNone            = "none"
	SensitivitySourceCode      = "source_code"
	SensitivityPII             = "pii"
	SensitivitySecretsPossible = "secrets_possible"
)

type RiskResult struct {
	RiskLevel   string
	Sensitivity string
	Reasons     []string
	Signals     []string
}

// ClassifyRisk derives risk from extracted features, the classified task type,
// and an optional client hint. Hints may escalate but never lower derived risk.
func ClassifyRisk(features Features, taskType string, clientRiskHint string) RiskResult {
	classifier := riskClassifier{
		result: RiskResult{
			RiskLevel:   RiskLow,
			Sensitivity: SensitivityNone,
		},
		seenReasons: make(map[string]struct{}),
	}

	hasCodeOrPath := features.RequiresCode || len(features.FilesTouched) > 0
	hasAuth := hasAnyKeyword(features.Keywords, "auth")
	hasPayment := hasAnyKeyword(features.Keywords, "payment")
	hasMigration := hasAnyKeyword(features.Keywords, "migration", "sql") || taskType == "database_migration"
	hasSecurity := hasAnyKeyword(features.Keywords, "security") || taskType == "security_review"
	hasProduction := hasAnyKeyword(features.Keywords, "production")
	hasUrgent := hasAnyKeyword(features.Keywords, "urgent")
	hasChangeIntent := hasAnyKeyword(features.Keywords, "change_intent")
	hasExploit := hasAnyKeyword(features.Keywords, "exploit")
	hasSecrets := hasSensitivity(features.SensitivityHints, SensitivitySecretsPossible, "secret")
	hasPII := hasSensitivity(features.SensitivityHints, SensitivityPII)

	switch taskType {
	case "security_review":
		classifier.raiseTo(RiskHigh, "task_security_review")
	case "database_migration":
		classifier.raiseTo(RiskHigh, "task_database_migration")
	case "unknown_high_risk":
		classifier.raiseTo(RiskHigh, "task_unknown_high_risk")
	case "hard_code_debugging", "simple_code_edit":
		classifier.raiseTo(RiskMedium, "task_code")
	case "long_context_analysis":
		classifier.raiseTo(RiskMedium, "task_long_context")
	}

	if hasCodeOrPath {
		classifier.raiseTo(RiskMedium, "code_or_path_signal")
	}

	if hasAuth {
		classifier.raiseTo(RiskMedium, "auth_signal")
		if hasCodeOrPath || hasProduction || hasChangeIntent {
			classifier.raiseTo(RiskHigh, "auth_high_risk_context")
		}
	}
	if hasPayment {
		classifier.raiseTo(RiskMedium, "payment_signal")
		if hasCodeOrPath || hasProduction || hasChangeIntent {
			classifier.raiseTo(RiskHigh, "payment_high_risk_context")
		}
	}
	if hasMigration {
		classifier.raiseTo(RiskHigh, "migration_signal")
	}
	if hasSecurity {
		classifier.raiseTo(RiskHigh, "security_signal")
	}

	if hasProduction || hasUrgent {
		classifier.addReason("production_or_urgent_signal")
		if hasCodeOrPath || hasAuth || hasPayment || hasMigration || hasSecurity {
			classifier.raiseOneStepUpToHigh("production_or_urgent_escalation")
		}
	}

	if hasSecrets {
		classifier.setSensitivity(SensitivitySecretsPossible, "sensitivity_secrets_possible")
		classifier.raiseTo(RiskHigh, "secret_signal")
	}
	if hasPII {
		classifier.setSensitivity(SensitivityPII, "sensitivity_pii")
		classifier.raiseTo(RiskMedium, "pii_signal")
	}
	if features.RequiresCode || hasSensitivity(features.SensitivityHints, SensitivitySourceCode, "auth", "payment", "security") {
		classifier.setSensitivity(SensitivitySourceCode, "sensitivity_source_code")
	}

	if hasSecurity && hasProduction && (hasSecrets || hasExploit || hasUrgent) {
		classifier.raiseTo(RiskCritical, "critical_security_production_signal")
	}
	if hasProduction && hasSecrets && (hasExploit || hasUrgent) {
		classifier.raiseTo(RiskCritical, "critical_secret_production_signal")
	}

	if hint, ok := normalizedRisk(clientRiskHint); ok {
		classifier.raiseTo(hint, "client_risk_hint")
	}

	if len(classifier.result.Reasons) == 0 {
		classifier.addReason("no_risk_signals")
	}

	return classifier.result
}

type riskClassifier struct {
	result      RiskResult
	seenReasons map[string]struct{}
}

func (c *riskClassifier) raiseTo(level string, reason string) {
	if riskRank(level) > riskRank(c.result.RiskLevel) {
		c.result.RiskLevel = level
	}
	c.addReason(reason)
}

func (c *riskClassifier) raiseOneStepUpToHigh(reason string) {
	switch c.result.RiskLevel {
	case RiskLow:
		c.result.RiskLevel = RiskMedium
	case RiskMedium:
		c.result.RiskLevel = RiskHigh
	}
	c.addReason(reason)
}

func (c *riskClassifier) setSensitivity(sensitivity string, reason string) {
	if sensitivityRank(sensitivity) > sensitivityRank(c.result.Sensitivity) {
		c.result.Sensitivity = sensitivity
	}
	c.addReason(reason)
}

func (c *riskClassifier) addReason(reason string) {
	if reason == "" {
		return
	}
	if _, ok := c.seenReasons[reason]; ok {
		return
	}
	c.seenReasons[reason] = struct{}{}
	c.result.Reasons = append(c.result.Reasons, reason)
	c.result.Signals = append(c.result.Signals, reason)
}

func normalizedRisk(value string) (string, bool) {
	switch value {
	case RiskLow, RiskMedium, RiskHigh, RiskCritical:
		return value, true
	default:
		return "", false
	}
}

func riskRank(level string) int {
	switch level {
	case RiskCritical:
		return 4
	case RiskHigh:
		return 3
	case RiskMedium:
		return 2
	case RiskLow:
		return 1
	default:
		return 0
	}
}

func sensitivityRank(sensitivity string) int {
	switch sensitivity {
	case SensitivitySecretsPossible:
		return 3
	case SensitivityPII:
		return 2
	case SensitivitySourceCode:
		return 1
	default:
		return 0
	}
}

func hasSensitivity(hints []string, wanted ...string) bool {
	for _, hint := range hints {
		for _, want := range wanted {
			if hint == want {
				return true
			}
		}
	}
	return false
}
