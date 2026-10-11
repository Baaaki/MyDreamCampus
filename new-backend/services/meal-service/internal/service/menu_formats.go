package service

import (
	"strings"
	"time"
)

// A month's menu is read two ways (see the seed's 06-meal.sql): the web
// keeps weeks, {"normalMenus": [week…], "veganMenus": [week…]}, week i being
// the i-th Monday-to-Sunday week touching the month; mobile looks a day up
// by date, {"YYYY-MM-DD": {"lunch": [...]}}. The web admin edits weeks only,
// so a save fills the dates in from them — otherwise every edit would empty
// the month on mobile.

var menuWeekdays = map[time.Weekday]string{
	time.Monday:    "monday",
	time.Tuesday:   "tuesday",
	time.Wednesday: "wednesday",
	time.Thursday:  "thursday",
	time.Friday:    "friday",
}

// withDatedLunches adds a date entry for each weekday the normal menu fills,
// its lunch being that day's normal menu as in the seed; the web format has
// no dinner to carry over. A date the request already holds is left as sent.
func withDatedLunches(menu map[string]any, year, month int) map[string]any {
	weeks, ok := menu["normalMenus"].([]any)
	if !ok {
		return menu
	}

	out := make(map[string]any, len(menu)+31)
	for k, v := range menu {
		out[k] = v
	}

	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	sinceMonday := (int(first.Weekday()) + 6) % 7
	for day := first; day.Month() == first.Month(); day = day.AddDate(0, 0, 1) {
		weekday, ok := menuWeekdays[day.Weekday()]
		if !ok {
			continue
		}
		key := day.Format("2006-01-02")
		if _, sent := out[key]; sent {
			continue
		}
		week := (day.Day() + sinceMonday - 1) / 7
		if week >= len(weeks) {
			continue
		}
		days, _ := weeks[week].(map[string]any)
		entry, _ := days[weekday].(map[string]any)
		if dishes := dishNames(entry["items"]); len(dishes) > 0 {
			out[key] = map[string]any{"lunch": dishes}
		}
	}
	return out
}

// dishNames keeps the filled-in names of a day's items; the editor leaves
// an unused course as an empty string.
func dishNames(items any) []any {
	raw, _ := items.([]any)
	names := make([]any, 0, len(raw))
	for _, item := range raw {
		if name, ok := item.(string); ok && strings.TrimSpace(name) != "" {
			names = append(names, name)
		}
	}
	return names
}

// isWeekMenu reports whether a save comes in the web's week format.
func isWeekMenu(menu map[string]any) bool {
	_, ok := menu["normalMenus"].([]any)
	return ok
}

// withDinnersKept carries the stored dinners into a week-format save. The
// web editor has no dinner to send, so without this an edit would drop the
// dinners mobile shows; a date the save already gives a dinner keeps its own.
func withDinnersKept(menu, stored map[string]any) map[string]any {
	out := make(map[string]any, len(menu))
	for k, v := range menu {
		out[k] = v
	}
	for key, value := range stored {
		day, _ := value.(map[string]any)
		dinner, ok := day["dinner"]
		if !ok {
			continue
		}
		entry, _ := out[key].(map[string]any)
		if _, own := entry["dinner"]; own {
			continue
		}
		merged := make(map[string]any, len(entry)+1)
		for k, v := range entry {
			merged[k] = v
		}
		merged["dinner"] = dinner
		out[key] = merged
	}
	return out
}
