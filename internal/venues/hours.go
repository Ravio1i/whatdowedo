package venues

import (
	"regexp"
	"strings"
	"time"
)

var dayNames = []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}

func dayIndex(t time.Time) int { // Monday = 0
	return (int(t.Weekday()) + 6) % 7
}

var timeRange = regexp.MustCompile(`\d{1,2}:\d{2}\s*-\s*\d{1,2}:\d{2}`)

// EvaluateOSMHours interprets a subset of the OSM opening_hours syntax and
// returns the status for the given date plus the matching time ranges.
// Supported: "24/7", "Mo-Fr 09:00-18:00; Sa 10:00-14:00; Su off", "Mo,We,Fr ...",
// "PH off" (treated as unknown unless isHoliday). Anything else -> unknown.
func EvaluateOSMHours(spec string, date time.Time, isHoliday bool) (OpenStatus, string) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return StatusUnknown, ""
	}
	if spec == "24/7" {
		return StatusOpen, "24/7"
	}
	today := dayIndex(date)
	status := StatusUnknown
	hours := ""
	matched := false
	skipped := false
	for _, rule := range strings.Split(spec, ";") {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		// Rules that cannot be evaluated: month/week/date-based conditions.
		if strings.ContainsAny(rule, "[]\"") || containsMonth(rule) {
			skipped = true
			continue
		}
		days, rest := splitDays(rule)
		applies := false
		if len(days) == 0 { // rule without day selector applies to every day
			applies = true
		} else {
			for _, d := range days {
				if d == today || (d == -1 && isHoliday) {
					applies = true
				}
			}
		}
		if !applies {
			continue
		}
		matched = true
		rest = strings.TrimSpace(rest)
		switch {
		case rest == "off" || rest == "closed":
			status, hours = StatusClosed, ""
		case rest == "" || rest == "open":
			status, hours = StatusOpen, "all day"
		default:
			if r := timeRange.FindAllString(rest, -1); len(r) > 0 {
				status, hours = StatusOpen, strings.Join(r, ", ")
			} else {
				status, hours = StatusUnknown, rest
			}
		}
	}
	if !matched {
		// Day selectors existed but none covered today -> closed by OSM semantics.
		if hasDaySelector(spec) && !skipped {
			return StatusClosed, ""
		}
		return StatusUnknown, ""
	}
	return status, hours
}

func containsMonth(s string) bool {
	for _, m := range []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"} {
		if strings.Contains(s, m+" ") || strings.HasPrefix(s, m) {
			return true
		}
	}
	return false
}

func hasDaySelector(spec string) bool {
	for _, d := range dayNames {
		if strings.Contains(spec, d) {
			return true
		}
	}
	return false
}

// splitDays parses a leading day selector ("Mo-Fr,Su", "PH") and returns the
// day indices (-1 for PH) and the remainder of the rule.
func splitDays(rule string) ([]int, string) {
	fields := strings.Fields(rule)
	if len(fields) == 0 {
		return nil, ""
	}
	sel := fields[0]
	if !looksLikeDaySelector(sel) {
		return nil, rule
	}
	var days []int
	for _, part := range strings.Split(sel, ",") {
		part = strings.TrimSpace(part)
		if part == "PH" {
			days = append(days, -1)
			continue
		}
		if strings.Contains(part, "-") {
			ab := strings.SplitN(part, "-", 2)
			a, b := idx(ab[0]), idx(ab[1])
			if a < 0 || b < 0 {
				continue
			}
			for d := a; ; d = (d + 1) % 7 {
				days = append(days, d)
				if d == b {
					break
				}
			}
			continue
		}
		if d := idx(part); d >= 0 {
			days = append(days, d)
		}
	}
	return days, strings.Join(fields[1:], " ")
}

func looksLikeDaySelector(s string) bool {
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "PH" || p == "SH" {
			continue
		}
		for _, part := range strings.Split(p, "-") {
			if idx(part) < 0 {
				return false
			}
		}
	}
	return true
}

func idx(d string) int {
	for i, n := range dayNames {
		if n == d {
			return i
		}
	}
	return -1
}
