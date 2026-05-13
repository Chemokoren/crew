package rbac

import (
	"context"
	"encoding/json"
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
}

// EvaluationResult holds the outcome of a policy evaluation.
type EvaluationResult struct {
	Allowed    bool
	DeniedBy   string // policy name that denied access (empty if allowed)
	Reason     string
	PolicyID   string
}

// Evaluate checks a list of policies against the given evaluation context.
// Policies are evaluated in priority order (highest first).
// A DENY policy overrides ALLOW. If no policies match, access is allowed (RBAC-granted).
func (pe *PolicyEngine) Evaluate(_ context.Context, policies []models.Policy, evalCtx EvaluationContext) EvaluationResult {
	if len(policies) == 0 {
		return EvaluationResult{Allowed: true}
	}

	for _, policy := range policies {
		if !policy.IsActive {
			continue
		}

		var conditions models.PolicyConditions
		if err := json.Unmarshal(policy.Conditions, &conditions); err != nil {
			continue // skip malformed policies
		}

		matches := pe.evaluateConditions(conditions, evalCtx)
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

// evaluateConditions checks if ALL conditions in a policy match the context.
func (pe *PolicyEngine) evaluateConditions(cond models.PolicyConditions, ctx EvaluationContext) bool {
	// Time range check
	if cond.TimeRange != nil {
		if !pe.checkTimeRange(*cond.TimeRange, ctx) {
			return false
		}
	}

	// MFA check
	if cond.MFARequired != nil && *cond.MFARequired && !ctx.MFAVerified {
		return true // condition matches: MFA is required but not verified → policy triggers
	}

	// Amount threshold check
	if cond.MaxAmount != nil && ctx.Amount > *cond.MaxAmount {
		return true // condition matches: amount exceeds limit → policy triggers
	}

	// IP allowlist check
	if len(cond.IPAllowList) > 0 {
		found := false
		for _, ip := range cond.IPAllowList {
			if ip == ctx.IPAddress {
				found = true
				break
			}
		}
		if !found {
			return true // condition matches: IP not in allowlist → policy triggers
		}
	}

	return false
}

// checkTimeRange verifies if current time falls outside the allowed window.
func (pe *PolicyEngine) checkTimeRange(tr models.TimeRangeCondition, ctx EvaluationContext) bool {
	tz := ctx.Timezone
	if tr.Timezone != "" {
		tz = tr.Timezone
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
