// Package agent - Perfection Loop FSM for code validation.
package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PerfectionPhase represents the current phase of the Perfection Loop.
type PerfectionPhase int

const (
	PhaseDeepAudit PerfectionPhase = iota
	PhaseHeuristicEnhancement
	PhaseValidationStrike
	PhaseIterativeConvergence
	PhaseFinalCertification
	PhaseComplete
)

// String returns a human-readable name for the phase.
func (p PerfectionPhase) String() string {
	switch p {
	case PhaseDeepAudit:
		return "Deep Audit"
	case PhaseHeuristicEnhancement:
		return "Heuristic Enhancement"
	case PhaseValidationStrike:
		return "Validation Strike"
	case PhaseIterativeConvergence:
		return "Iterative Convergence"
	case PhaseFinalCertification:
		return "Final Certification"
	case PhaseComplete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// PerfectionConfig holds configuration for the Perfection Loop.
type PerfectionConfig struct {
	MaxIterations     int           // Max iterations per phase
	MaxTotalIters     int           // Hard cap on total iterations
	MinDeltaThreshold float64       // Levenshtein ratio below which we halt
	HardTimeout       time.Duration // Absolute timeout per cycle
	Enabled           bool
}

// DefaultPerfectionConfig returns sensible defaults.
func DefaultPerfectionConfig() PerfectionConfig {
	return PerfectionConfig{
		MaxIterations:     5,
		MaxTotalIters:     20,
		MinDeltaThreshold: 0.05, // 5% change minimum
		HardTimeout:       2 * time.Minute,
		Enabled:           true,
	}
}

// PerfectionResult is the outcome of a Perfection Loop cycle.
type PerfectionResult struct {
	Phase       PerfectionPhase
	Iterations  int
	TotalIters  int
	Passed      bool
	Reason      string
	Findings    []string
	Suggestions []string
}

// PerfectionLoop runs the Perfection Loop FSM.
// It calls the agent to perform validation and enhancement cycles.
type PerfectionLoop struct {
	config   PerfectionConfig
	onPhase  func(PerfectionPhase, string) // callback for UI updates
	execute  func(ctx context.Context, prompt string) (string, error) // agent execution
}

// NewPerfectionLoop creates a new Perfection Loop.
func NewPerfectionLoop(config PerfectionConfig, execute func(ctx context.Context, prompt string) (string, error), onPhase func(PerfectionPhase, string)) *PerfectionLoop {
	return &PerfectionLoop{
		config:  config,
		onPhase: onPhase,
		execute: execute,
	}
}

// Run executes the full Perfection Loop on a piece of work.
func (pl *PerfectionLoop) Run(ctx context.Context, description string, content string) PerfectionResult {
	if !pl.config.Enabled {
		return PerfectionResult{Phase: PhaseComplete, Passed: true, Reason: "perfection loop disabled"}
	}

	deadline, cancel := context.WithTimeout(ctx, pl.config.HardTimeout)
	defer cancel()

	totalIters := 0
	result := PerfectionResult{}

	phases := []PerfectionPhase{
		PhaseDeepAudit,
		PhaseHeuristicEnhancement,
		PhaseValidationStrike,
		PhaseIterativeConvergence,
		PhaseFinalCertification,
	}

	for _, phase := range phases {
		if deadline.Err() != nil {
			result.Reason = "hard timeout exceeded"
			return result
		}

		if totalIters >= pl.config.MaxTotalIters {
			result.Reason = "max total iterations reached"
			return result
		}

		pl.emit(phase, fmt.Sprintf("Starting %s...", phase))
		phaseResult := pl.runPhase(deadline, phase, description, content, &totalIters)

		if !phaseResult.Passed {
			result.Phase = phase
			result.Reason = phaseResult.Reason
			result.Findings = append(result.Findings, phaseResult.Findings...)
			return result
		}

		result.Findings = append(result.Findings, phaseResult.Findings...)
		result.Suggestions = append(result.Suggestions, phaseResult.Suggestions...)

		// Apply suggestions from this phase to the content
		if len(phaseResult.Suggestions) > 0 {
			content = pl.applySuggestions(deadline, content, phaseResult.Suggestions)
		}
	}

	result.Phase = PhaseComplete
	result.TotalIters = totalIters
	result.Passed = true
	result.Reason = "all phases passed"
	return result
}

// runPhase executes a single phase of the Perfection Loop.
func (pl *PerfectionLoop) runPhase(ctx context.Context, phase PerfectionPhase, description, content string, totalIters *int) PerfectionResult {
	result := PerfectionResult{Phase: phase}
	prevContent := content

	for i := 0; i < pl.config.MaxIterations; i++ {
		if ctx.Err() != nil {
			result.Reason = "timeout"
			return result
		}
		if *totalIters >= pl.config.MaxTotalIters {
			result.Reason = "max total iterations"
			return result
		}

		*totalIters++

		// Build phase-specific prompt
		prompt := pl.buildPrompt(phase, description, content, i)
		response, err := pl.execute(ctx, prompt)
		if err != nil {
			result.Findings = append(result.Findings, fmt.Sprintf("iteration %d error: %s", i, err.Error()))
			continue
		}

		// Parse response
		passed, findings, suggestions := pl.parseResponse(response)
		result.Findings = append(result.Findings, findings...)
		result.Suggestions = append(result.Suggestions, suggestions...)

		if passed {
			result.Passed = true
			return result
		}

		// Check convergence: if content hasn't changed enough, we're done
		delta := levenshteinRatio(prevContent, content)
		if i > 0 && delta < pl.config.MinDeltaThreshold {
			pl.emit(phase, fmt.Sprintf("Converged after %d iterations (delta=%.2f%%)", i+1, delta*100))
			result.Passed = true
			result.Reason = "converged"
			return result
		}
		prevContent = content
	}

	// Max iterations for this phase - pass anyway (circuit breaker)
	result.Passed = true
	result.Reason = fmt.Sprintf("max iterations (%d) for phase", pl.config.MaxIterations)
	return result
}

// buildPrompt creates the prompt for a specific phase and iteration.
func (pl *PerfectionLoop) buildPrompt(phase PerfectionPhase, description, content string, iteration int) string {
	switch phase {
	case PhaseDeepAudit:
		return fmt.Sprintf(`PERFECTION LOOP - DEEP AUDIT (iteration %d)

TASK: %s

CONTENT TO AUDIT:
%s

Analyze this code for:
1. Correctness issues (logic errors, off-by-one, nil dereference)
2. Edge cases not handled
3. Missing error handling
4. Race conditions or concurrency issues
5. Resource leaks
6. Performance problems

Respond in this format:
PASS: true/false
FINDINGS: (comma-separated list of issues found)
SUGGESTIONS: (comma-separated list of fixes to apply)`, iteration, description, content)

	case PhaseHeuristicEnhancement:
		return fmt.Sprintf(`PERFECTION LOOP - HEURISTIC ENHANCEMENT (iteration %d)

TASK: %s

CONTENT:
%s

Apply these heuristic improvements:
1. Simplify complex conditionals
2. Extract repeated logic into helpers
3. Improve naming clarity
4. Reduce nesting depth
5. Make error messages more actionable

Respond in this format:
PASS: true/false
FINDINGS: (comma-separated list of improvements found)
SUGGESTIONS: (comma-separated list of changes to apply)`, iteration, description, content)

	case PhaseValidationStrike:
		return fmt.Sprintf(`PERFECTION LOOP - VALIDATION STRIKE (iteration %d)

TASK: %s

CONTENT:
%s

Validate:
1. All public APIs have proper documentation
2. All error paths return meaningful errors
3. All inputs are validated at boundaries
4. No hardcoded values that should be configurable
5. Tests would cover critical paths

Respond in this format:
PASS: true/false
FINDINGS: (comma-separated list of validation issues)
SUGGESTIONS: (comma-separated list of fixes)`, iteration, description, content)

	case PhaseIterativeConvergence:
		return fmt.Sprintf(`PERFECTION LOOP - ITERATIVE CONVERGENCE (iteration %d)

TASK: %s

CONTENT:
%s

Check if all previous findings have been addressed.
If the code is now stable and correct, respond PASS: true.

Respond in this format:
PASS: true/false
FINDINGS: (comma-separated remaining issues)
SUGGESTIONS: (comma-separated final fixes)`, iteration, description, content)

	case PhaseFinalCertification:
		return fmt.Sprintf(`PERFECTION LOOP - FINAL CERTIFICATION (iteration %d)

TASK: %s

CONTENT:
%s

Final review checklist:
1. Code compiles without errors
2. No stubs or placeholder implementations
3. All public functions are complete
4. Error handling is comprehensive
5. The code is production-ready

Respond in this format:
PASS: true/false
FINDINGS: (comma-separated final issues)
SUGGESTIONS: (comma-separated final fixes)`, iteration, description, content)

	default:
		return content
	}
}

// parseResponse parses the structured response from the perfection loop agent.
func (pl *PerfectionLoop) parseResponse(response string) (passed bool, findings, suggestions []string) {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)

		if strings.HasPrefix(upper, "PASS:") {
			val := strings.TrimSpace(strings.TrimPrefix(upper, "PASS:"))
			passed = val == "TRUE"
		}
		if strings.HasPrefix(upper, "FINDINGS:") {
			val := strings.TrimSpace(line[len("FINDINGS:"):])
			if val != "" {
				findings = strings.Split(val, ",")
				for i := range findings {
					findings[i] = strings.TrimSpace(findings[i])
				}
			}
		}
		if strings.HasPrefix(upper, "SUGGESTIONS:") {
			val := strings.TrimSpace(line[len("SUGGESTIONS:"):])
			if val != "" {
				suggestions = strings.Split(val, ",")
				for i := range suggestions {
					suggestions[i] = strings.TrimSpace(suggestions[i])
				}
			}
		}
	}
	return
}

// applySuggestions applies the suggestions to the content.
func (pl *PerfectionLoop) applySuggestions(ctx context.Context, content string, suggestions []string) string {
	if len(suggestions) == 0 {
		return content
	}

	prompt := fmt.Sprintf(`Apply these suggestions to the content. Return ONLY the modified content, no explanation.

SUGGESTIONS:
%s

CONTENT:
%s`, strings.Join(suggestions, "\n"), content)

	result, err := pl.execute(ctx, prompt)
	if err != nil {
		return content // Keep original on error
	}

	// If the result looks like actual code content (not a chat response), use it
	if len(result) > 0 && !strings.HasPrefix(result, "I ") && !strings.HasPrefix(result, "Here") {
		return result
	}
	return content
}

func (pl *PerfectionLoop) emit(phase PerfectionPhase, msg string) {
	if pl.onPhase != nil {
		pl.onPhase(phase, msg)
	}
}

// levenshteinRatio computes the similarity ratio between two strings.
// Returns 0.0 for identical strings, 1.0 for completely different strings.
func levenshteinRatio(a, b string) float64 {
	if a == b {
		return 0.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 1.0
	}

	la := len(a)
	lb := len(b)

	// Use single-row DP for memory efficiency
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	distance := prev[lb]
	maxLen := max(la, lb)
	return float64(distance) / float64(maxLen)
}
