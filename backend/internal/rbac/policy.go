package rbac

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/kibsoft/amy-mis/internal/models"
)

// PolicyEngine evaluates dynamic policy conditions on top of RBAC permissions.
type PolicyEngine struct{}

// NewPolicyEngine creates a new policy engine instance.
func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{}
}

// EvaluationContext provides runtime context for policy evaluation.
type EvaluationContext struct {
	CurrentTime time.Time
	IPAddress   string
	MFAVerified bool
	Amount      int64  // monetary amount in cents (for financial policies)
	DeviceType  string // "web", "mobile", "ussd"
	Timezone    string // e.g. "Africa/Nairobi"
	RiskLevel   string // low, medium, high, critical
}

// EvaluationResult holds the outcome of a policy evaluation.
type EvaluationResult struct {
	Allowed  bool
	DeniedBy string // policy name that denied access (empty if allowed)
	Reason   string
	PolicyID string
}

// Evaluate checks a list of policies against the given evaluation context.
// Policies are evaluated in priority order (highest first).
// A DENY policy overrides ALLOW. If no policies match, access is allowed (RBAC-granted).
func (pe *PolicyEngine) Evaluate(_ context.Context, policies []models.Policy, evalCtx EvaluationContext) EvaluationResult {
	if len(policies) == 0 {
		return EvaluationResult{Allowed: true}
	}
	if evalCtx.CurrentTime.IsZero() {
		evalCtx.CurrentTime = time.Now()
	}

	ordered := append([]models.Policy(nil), policies...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Priority > ordered[j].Priority
	})

	for _, policy := range ordered {
		if !policy.IsActive {
			continue
		}

		matches, err := pe.evaluateConditionJSON(policy.Conditions, evalCtx)
		if err != nil {
			continue // skip malformed policies
		}
		if !matches {
			continue
		}

		if policy.Effect == models.PolicyEffectDeny {
			return EvaluationResult{
				Allowed:  false,
				DeniedBy: policy.Name,
				Reason:   policy.Description,
				PolicyID: policy.ID.String(),
			}
		}
	}

	return EvaluationResult{Allowed: true}
}

// evaluateConditionJSON supports both flat condition objects and simple
// composable trees: {"and": [...]}, {"or": [...]}, {"not": {...}}.
func (pe *PolicyEngine) evaluateConditionJSON(raw json.RawMessage, ctx EvaluationContext) (bool, error) {
	if len(raw) == 0 || string(raw) == "{}" {
		return true, nil
	}

	var node map[string]json.RawMessage
	if err := json.Unmarshal(raw, &node); err != nil {
		return false, err
	}
	if len(node) == 0 {
		return true, nil
	}

	if childRaw, ok := node["and"]; ok {
		var children []json.RawMessage
		if err := json.Unmarshal(childRaw, &children); err != nil {
			return false, err
		}
		for _, child := range children {
			matches, err := pe.evaluateConditionJSON(child, ctx)
			if err != nil || !matches {
				return false, err
			}
		}
		return true, nil
	}

	if childRaw, ok := node["or"]; ok {
		var children []json.RawMessage
		if err := json.Unmarshal(childRaw, &children); err != nil {
			return false, err
		}
		for _, child := range children {
			matches, err := pe.evaluateConditionJSON(child, ctx)
			if err != nil {
				return false, err
			}
			if matches {
				return true, nil
			}
		}
		return false, nil
	}

	if childRaw, ok := node["not"]; ok {
		matches, err := pe.evaluateConditionJSON(childRaw, ctx)
		return !matches, err
	}

	return pe.evaluateFlatConditions(raw, ctx)
}

// evaluateFlatConditions checks if all provided flat conditions match the context.
func (pe *PolicyEngine) evaluateFlatConditions(raw json.RawMessage, ctx EvaluationContext) (bool, error) {
	var cond models.PolicyConditions
	if err := json.Unmarshal(raw, &cond); err != nil {
		return false, err
	}

	provided := 0
	matches := true

	if cond.TimeRange != nil {
		provided++
		matches = matches && pe.checkTimeRange(*cond.TimeRange, ctx)
	}

	if cond.MFARequired != nil {
		provided++
		matches = matches && *cond.MFARequired && !ctx.MFAVerified
	}

	if cond.MaxAmount != nil {
		provided++
		matches = matches && ctx.Amount > *cond.MaxAmount
	}

	var extra struct {
		AmountThreshold *int64 `json:"amount_threshold"`
	}
	if err := json.Unmarshal(raw, &extra); err == nil && extra.AmountThreshold != nil {
		provided++
		matches = matches && ctx.Amount > *extra.AmountThreshold
	}

	if cond.RequiredRiskLevel != "" {
		provided++
		matches = matches && riskAtLeast(ctx.RiskLevel, cond.RequiredRiskLevel)
	}

	if len(cond.IPAllowList) > 0 {
		provided++
		found := false
		for _, ip := range cond.IPAllowList {
			if ip == ctx.IPAddress {
				found = true
				break
			}
		}
		matches = matches && !found
	}

	if provided == 0 {
		return true, nil
	}
	return matches, nil
}

// checkTimeRange verifies if current time falls outside the allowed window.
func (pe *PolicyEngine) checkTimeRange(tr models.TimeRangeCondition, ctx EvaluationContext) bool {
	tz := ctx.Timezone
	if tr.Timezone != "" {
		tz = tr.Timezone
	}
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	now := ctx.CurrentTime.In(loc)
	hour := now.Hour()
	weekday := int(now.Weekday())

	// Check day-of-week restriction
	if len(tr.DaysOfWeek) > 0 {
		dayAllowed := false
		for _, d := range tr.DaysOfWeek {
			if d == weekday {
				dayAllowed = true
				break
			}
		}
		if !dayAllowed {
			return true // outside allowed days → condition matches
		}
	}

	// Check hour range
	if tr.StartHour <= tr.EndHour {
		// Normal range (e.g., 8-17)
		if hour < tr.StartHour || hour >= tr.EndHour {
			return true // outside business hours → condition matches
		}
	} else {
		// Overnight range (e.g., 22-6)
		if hour < tr.StartHour && hour >= tr.EndHour {
			return true // outside overnight window → condition matches
		}
	}

	return false
}

func riskAtLeast(actual, required string) bool {
	ranks := map[string]int{
		models.RiskLow:      1,
		models.RiskMedium:   2,
		models.RiskHigh:     3,
		models.RiskCritical: 4,
	}
	actualRank, ok := ranks[actual]
	if !ok {
		return false
	}
	requiredRank, ok := ranks[required]
	if !ok {
		return false
	}
	return actualRank >= requiredRank
}
