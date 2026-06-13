// Package template provides template resolution and argument coercion for execution plans.
package template

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// templateRe extracts the content inside {{...}} delimiters.
// The actual step name / field resolution is done in resolveTemplateContent
// to handle dots in MCP step names (e.g., {{mcp.his.query_patient}}).
var templateRe = regexp.MustCompile(`\{\{([\w][\w.-]*)\}\}`)

// Resolve replaces {{step_name}} and {{step_name.field}} references in a string
// with values from results. A single template spanning the entire string preserves
// the original type; embedded templates become strings.
//
// Returns an error when a step name or field path cannot be resolved — there is
// no silent fallback. The error message lists available step names / field keys
// to help the caller fix the template.
func Resolve(s string, results map[string]any) (any, error) {
	if !strings.Contains(s, "{{") || len(results) == 0 {
		return s, nil
	}
	matches := templateRe.FindAllStringSubmatchIndex(s, -1)
	if len(matches) == 0 {
		return s, nil
	}

	// Single template spanning the entire string → preserve type
	if len(matches) == 1 {
		m := matches[0]
		fullStart, fullEnd := m[0], m[1]
		if fullStart == 0 && fullEnd == len(s) {
			content := s[m[2]:m[3]]
			stepName, field := resolveTemplateContent(content, results)
			res, ok := results[stepName]
			if !ok {
				return nil, fmt.Errorf("template: step %q not found (available: %s)",
					stepName, strings.Join(resultKeys(results), ", "))
			}
			if field != "" {
				v, err := extractField(res, field)
				if err != nil {
					return nil, fmt.Errorf("template %q: %w", s, err)
				}
				return stringifyComplex(v), nil
			}
			return stringifyComplex(res), nil
		}
	}

	// Multiple or embedded templates → string interpolation
	var errs []string
	result := templateRe.ReplaceAllStringFunc(s, func(m string) string {
		parts := templateRe.FindStringSubmatch(m)
		content := parts[1]
		stepName, field := resolveTemplateContent(content, results)
		res, ok := results[stepName]
		if !ok {
			errs = append(errs, fmt.Sprintf("step %q not found", stepName))
			return m
		}
		if field != "" {
			v, err := extractField(res, field)
			if err != nil {
				errs = append(errs, err.Error())
				return m
			}
			return stringifyValue(v)
		}
		return stringifyValue(res)
	})
	if len(errs) > 0 {
		return nil, fmt.Errorf("template %q: %s", s, strings.Join(errs, "; "))
	}
	return result, nil
}

// resolveTemplateContent splits a template's inner content into stepName and
// optional field, using the results map to resolve ambiguity when step names
// contain dots (e.g., mcp.his.query_patient).
func resolveTemplateContent(content string, results map[string]any) (stepName, field string) {
	// Try full content as step name first (handles {{mcp.his.query_patient}})
	if _, ok := results[content]; ok {
		return content, ""
	}
	// Split from right: last segment is field, rest is step name
	if !strings.Contains(content, ".") {
		return content, ""
	}
	lastDot := strings.LastIndex(content, ".")
	candidate := content[:lastDot]
	candidateField := content[lastDot+1:]
	if _, ok := results[candidate]; ok {
		return candidate, candidateField
	}
	// Try progressively shorter prefixes
	for {
		dot := strings.LastIndex(candidate, ".")
		if dot < 0 {
			break
		}
		candidateField = candidate[dot+1:] + "." + candidateField
		candidate = candidate[:dot]
		if _, ok := results[candidate]; ok {
			return candidate, candidateField
		}
	}
	return content, ""
}

// CoerceStringNumbers converts string values in args that represent numbers
// (integer, float, or boolean) into their native types. This handles the common
// case where an LLM generates {"a": "42"} instead of {"a": 42}.
// Returns the number of values that were coerced.
func CoerceStringNumbers(args map[string]any) int {
	coerced := 0
	for key, val := range args {
		strVal, ok := val.(string)
		if !ok {
			continue
		}
		if v, ok := tryParseInt(strVal); ok {
			args[key] = v
			coerced++
		} else if v, ok := tryParseFloat(strVal); ok {
			args[key] = v
			coerced++
		} else if v, ok := tryParseBool(strVal); ok {
			args[key] = v
			coerced++
		}
	}
	return coerced
}

// extractField walks a dotted field path into a result map and returns the value.
// Returns an error listing available keys when a segment cannot be found.
func extractField(res any, field string) (any, error) {
	current := res
	path := strings.Split(field, ".")
	for i, f := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("field %q: cannot index into %T at %q",
				field, current, strings.Join(path[:i], "."))
		}
		v, exists := m[f]
		if !exists {
			return nil, fmt.Errorf("field %q not found (available: %s)",
				f, strings.Join(mapKeys(m), ", "))
		}
		current = v
	}
	return current, nil
}

// resultKeys returns sorted step names from the results map.
func resultKeys(results map[string]any) []string {
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// mapKeys returns sorted keys from a map.
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func stringifyComplex(v any) any {
	switch v := v.(type) {
	case string:
		return v
	case map[string]any, []any:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// stringifyValue converts any value to a string for template interpolation.
// Maps and slices are JSON-serialized; everything else uses fmt.Sprintf.
func stringifyValue(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case map[string]any, []any:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func tryParseInt(s string) (int64, bool) {
	if len(s) == 0 {
		return 0, false
	}
	for i, c := range s {
		if c == '-' && i == 0 {
			continue
		}
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	var n int64
	for _, c := range s {
		if c == '-' {
			continue
		}
		n = n*10 + int64(c-'0')
	}
	if s[0] == '-' {
		n = -n
	}
	return n, true
}

func tryParseFloat(s string) (float64, bool) {
	if len(s) == 0 {
		return 0, false
	}
	hasDot := false
	for i, c := range s {
		if c == '-' && i == 0 {
			continue
		}
		if c == '.' {
			if hasDot {
				return 0, false
			}
			hasDot = true
			continue
		}
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	if !hasDot {
		return 0, false
	}
	var neg bool
	i := 0
	if s[0] == '-' {
		neg = true
		i = 1
	}
	var val float64
	for ; i < len(s); i++ {
		if s[i] == '.' {
			break
		}
		val = val*10 + float64(s[i]-'0')
	}
	i++ // skip dot
	var frac float64
	var div float64 = 10
	for ; i < len(s); i++ {
		frac += float64(s[i]-'0') / div
		div *= 10
	}
	val += frac
	if neg {
		val = -val
	}
	return val, true
}

func tryParseBool(s string) (bool, bool) {
	if s == "true" {
		return true, true
	}
	if s == "false" {
		return false, true
	}
	return false, false
}
