package host

import "fmt"

// FixPlan is an ordered list of remediation steps derived from failed checks.
type FixPlan struct {
	Steps []PlannedFix
}

// PlannedFix pairs a CheckResult with its suggested fix.
type PlannedFix struct {
	Check CheckResult
	Step  FixStep
}

// Empty reports whether the fix plan has any steps.
func (p FixPlan) Empty() bool {
	return len(p.Steps) == 0
}

// AutomatableCount returns how many steps can be run non-interactively.
func (p FixPlan) AutomatableCount() int {
	n := 0
	for _, s := range p.Steps {
		if !s.Step.ManualOnly {
			n++
		}
	}
	return n
}

// BuildFixPlan produces a FixPlan from check results, including only checks
// with an associated FixStep.
func BuildFixPlan(results []CheckResult) FixPlan {
	var plan FixPlan
	for _, r := range results {
		if r.Severity == SeverityOK || r.Fix == nil {
			continue
		}
		plan.Steps = append(plan.Steps, PlannedFix{Check: r, Step: *r.Fix})
	}
	return plan
}

// Describe returns a short human-readable description of the fix plan.
func (p FixPlan) Describe() string {
	if p.Empty() {
		return "nothing to fix"
	}
	auto := p.AutomatableCount()
	manual := len(p.Steps) - auto
	switch {
	case auto > 0 && manual > 0:
		return fmt.Sprintf("%d fix(es) can run automatically, %d require manual steps", auto, manual)
	case auto > 0:
		return fmt.Sprintf("%d fix(es) can run automatically", auto)
	default:
		return fmt.Sprintf("%d fix(es) require manual steps", manual)
	}
}
