package tools

import (
	"regexp"
	"strings"
	"unicode"
)

// QualityGate validates forged tools before registration.
type QualityGate struct {
	stubRe     *regexp.Regexp
	namingRe   *regexp.Regexp
	actionable *regexp.Regexp
	versionRe  *regexp.Regexp
}

// QualityResult is the outcome of a quality check.
type QualityResult struct {
	Passed   bool              `json:"passed"`
	Failures []QualityFailure  `json:"failures,omitempty"`
}

// QualityFailure describes a single validation failure.
type QualityFailure struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// NewQualityGate creates a quality gate with pre-compiled regex.
func NewQualityGate() *QualityGate {
	return &QualityGate{
		stubRe:     regexp.MustCompile(`(?i)(//\s*todo|//\s*fixme|panic\("not implemented"\)|placeholder|\[STUB\]|__STUB__|TBD)`),
		namingRe:   regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`),
		actionable: regexp.MustCompile(`(?m)^\s*(\d+\.\s|[-*]\s|` + "`" + "{3})"),
		versionRe:  regexp.MustCompile(`^\d+\.\d+\.\d+$`),
	}
}

// Validate checks a forged tool against all quality gates.
func (qg *QualityGate) Validate(name, description, version, body string, existingNames map[string]bool) QualityResult {
	var failures []QualityFailure

	// Name required
	if strings.TrimSpace(name) == "" {
		failures = append(failures, QualityFailure{Code: "E_NAME_REQUIRED", Detail: "Skill name is empty"})
	} else if len(name) > 64 || !qg.namingRe.MatchString(name) {
		failures = append(failures, QualityFailure{Code: "E_NAME_FORMAT", Detail: "'" + name + "' must be kebab-case, start with a letter, max 64 chars"})
	}

	// Duplicate name
	nameLower := strings.ToLower(name)
	if existingNames[nameLower] {
		failures = append(failures, QualityFailure{Code: "E_DUPLICATE_NAME", Detail: "A tool named '" + name + "' already exists"})
	}

	// Similar name detection
	for existing := range existingNames {
		if keywordOverlap(name, description, existing) > 0.8 {
			failures = append(failures, QualityFailure{Code: "E_DUPLICATE_SIMILAR", Detail: "Tool is similar to existing tool '" + existing + "'. Consider patching that tool instead."})
			break
		}
	}

	// Description minimum length
	if len(strings.TrimSpace(description)) < 10 {
		failures = append(failures, QualityFailure{Code: "E_DESC_REQUIRED", Detail: "Description is too short (minimum 10 characters)"})
	}

	// Version format
	if strings.TrimSpace(version) == "" || !qg.versionRe.MatchString(version) {
		failures = append(failures, QualityFailure{Code: "E_VERSION_REQUIRED", Detail: "Version must match X.Y.Z (semver)"})
	}

	// No stubs
	if qg.stubRe.MatchString(body) {
		m := qg.stubRe.FindString(body)
		failures = append(failures, QualityFailure{Code: "E_STUBS_FOUND", Detail: "Body contains a stub: '" + m + "'"})
	}

	// Body length (excluding frontmatter)
	cleanBody := stripFrontmatter(body)
	if len(cleanBody) < 200 {
		failures = append(failures, QualityFailure{Code: "E_BODY_TOO_SHORT", Detail: "Body is too short (minimum 200 characters)"})
	}

	// Actionable content (must have list or code block)
	if !qg.actionable.MatchString(cleanBody) {
		failures = append(failures, QualityFailure{Code: "E_NO_ACTIONABLE", Detail: "No numbered list, bullet list, or code block found"})
	}

	return QualityResult{
		Passed:   len(failures) == 0,
		Failures: failures,
	}
}

func stripFrontmatter(body string) string {
	trimmed := strings.TrimSpace(body)
	if strings.HasPrefix(trimmed, "---") {
		rest := strings.TrimPrefix(trimmed, "---")
		if idx := strings.Index(rest, "---"); idx >= 0 {
			return strings.TrimSpace(rest[idx+3:])
		}
	}
	return trimmed
}

func keywordOverlap(name, description, existing string) float64 {
	combined := strings.ToLower(name + " " + description)
	wordsA := make(map[string]bool)
	for _, w := range strings.Fields(combined) {
		if len(w) > 2 && !isStopWord(w) {
			wordsA[w] = true
		}
	}

	wordsB := make(map[string]bool)
	for _, w := range strings.FieldsFunc(existing, func(r rune) bool { return r == '-' || r == ' ' || r == '_' }) {
		w = strings.ToLower(w)
		if len(w) > 2 && !isStopWord(w) {
			wordsB[w] = true
		}
	}

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	intersection := 0
	for w := range wordsA {
		if wordsB[w] {
			intersection++
		}
	}

	minLen := len(wordsA)
	if len(wordsB) < minLen {
		minLen = len(wordsB)
	}

	return float64(intersection) / float64(minLen)
}

func isStopWord(w string) bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"for": true, "to": true, "of": true, "in": true, "on": true,
		"is": true, "it": true, "by": true, "as": true, "at": true,
	}
	return stopWords[w]
}

// isKebabCase checks if a string is valid kebab-case.
func isKebabCase(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '-' {
			if i == 0 || i == len(s)-1 {
				return false
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
		if unicode.IsUpper(r) {
			return false
		}
	}
	return true
}
