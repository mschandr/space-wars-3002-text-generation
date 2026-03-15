package validation

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// lineNumberPrefix matches patterns like "Line 1:", "1.", "1)" at the start of a line.
var lineNumberPrefix = regexp.MustCompile(`(?i)^(line\s+)?\d+[\.\)\:]\s`)

// metaCommentaryPatterns are substrings that indicate the LLM leaked its reasoning.
var metaCommentaryPatterns = []string{
	"here are",
	"here's",
	"certainly",
	"of course",
	"as requested",
	"i'll generate",
	"sure, here",
	"sure here",
	"[system]",
	"<thinking>",
	"{thinking}",
	"```",
	"json",
}

// Validate filters a slice of raw LLM lines into accepted and rejected sets.
// Returns an error if fewer than 60% of lines pass — the caller should retry.
func Validate(lines []string) ([]string, []string, error) {
	var accepted []string
	var rejected []string
	seen := make(map[string]bool)

	for _, line := range lines {
		normalized := normalizeLine(line)

		if err := validateLine(normalized); err != nil {
			rejected = append(rejected, fmt.Sprintf("%s (reason: %v)", line, err))
			continue
		}

		// Deduplicate by normalized content hash
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(normalized)))
		if seen[hash] {
			rejected = append(rejected, fmt.Sprintf("%s (reason: duplicate)", line))
			continue
		}

		seen[hash] = true
		accepted = append(accepted, normalized)
	}

	// Require at least 60% of input lines to pass
	threshold := float64(len(lines)) * 0.6
	if float64(len(accepted)) < threshold {
		return nil, rejected, fmt.Errorf(
			"validation failed: only %d/%d lines passed (need at least %.0f)",
			len(accepted),
			len(lines),
			threshold,
		)
	}

	return accepted, rejected, nil
}

// normalizeLine trims whitespace, collapses runs of spaces, strips numbering
// prefixes, and removes duplicate terminal punctuation.
func normalizeLine(line string) string {
	// Trim surrounding whitespace
	line = strings.TrimSpace(line)

	// Collapse internal whitespace runs
	space := regexp.MustCompile(`\s+`)
	line = space.ReplaceAllString(line, " ")

	// Strip leading numbering prefixes like "1. ", "2) ", "Line 3: "
	line = lineNumberPrefix.ReplaceAllString(line, "")
	line = strings.TrimSpace(line)

	// Remove duplicate terminal punctuation (.. !! ??)
	for strings.HasSuffix(line, "..") ||
		strings.HasSuffix(line, "!!") ||
		strings.HasSuffix(line, "??") {
		line = line[:len(line)-1]
	}

	return line
}

// validateLine checks a single normalized line against all rejection criteria.
func validateLine(line string) error {
	if len(line) == 0 {
		return fmt.Errorf("empty line")
	}

	// Character limit
	if len(line) > 255 {
		return fmt.Errorf("too long: %d chars (max 255)", len(line))
	}

	// Word count
	words := strings.Fields(line)
	if len(words) < 6 {
		return fmt.Errorf("too few words: %d (min 6)", len(words))
	}
	if len(words) > 20 {
		return fmt.Errorf("too many words: %d (max 20)", len(words))
	}

	// No control characters (allow tab/newline/CR for normalization safety,
	// but those would have been collapsed by normalizeLine already)
	for _, r := range line {
		if unicode.IsControl(r) {
			return fmt.Errorf("contains control character: U+%04X", r)
		}
	}

	// No excessive punctuation
	if strings.Contains(line, "!!!") || strings.Contains(line, "???") {
		return fmt.Errorf("excessive punctuation")
	}

	// Meta-commentary / LLM leakage detection
	lower := strings.ToLower(line)
	for _, pattern := range metaCommentaryPatterns {
		if strings.Contains(lower, pattern) {
			return fmt.Errorf("meta commentary detected: matched %q", pattern)
		}
	}

	// Line-number prefix remnant (catches "1. blah" that survived normalization)
	if lineNumberPrefix.MatchString(line) {
		return fmt.Errorf("line number prefix detected")
	}

	return nil
}
