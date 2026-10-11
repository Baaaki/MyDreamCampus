package service

import (
	"context"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
	"github.com/baaaki/mydreamcampus/meal/internal/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func webDay(items ...any) map[string]any {
	return map[string]any{"items": items, "calories": 800}
}

// 1 Oct 2026 is a Thursday: week 0 runs 28 Sep - 4 Oct, week 2 opens 12 Oct.
func octoberWeeks() map[string]any {
	return map[string]any{
		"normalMenus": []any{
			map[string]any{"thursday": webDay("Mercimek Çorbası", "", "Pilav"), "friday": webDay("", "")},
			map[string]any{},
			map[string]any{"monday": webDay("Etli Nohut")},
		},
		"veganMenus": []any{},
	}
}

func TestWithDatedLunches_WebWeeks_DatesEachFilledWeekday(t *testing.T) {
	menu := withDatedLunches(octoberWeeks(), 2026, 10)

	assert.Equal(t, map[string]any{"lunch": []any{"Mercimek Çorbası", "Pilav"}}, menu["2026-10-01"])
	assert.Equal(t, map[string]any{"lunch": []any{"Etli Nohut"}}, menu["2026-10-12"])
	assert.NotContains(t, menu, "2026-10-02", "a day with no dishes has no entry")
	assert.NotContains(t, menu, "2026-10-26", "a week the menu leaves out has no entry")
	assert.Contains(t, menu, "normalMenus", "the web's weeks are kept")
}

func TestWithDatedLunches_DateAlreadySent_KeptAsSent(t *testing.T) {
	menu := octoberWeeks()
	sent := map[string]any{"lunch": []any{"Balık"}, "dinner": []any{"Köfte"}}
	menu["2026-10-01"] = sent

	assert.Equal(t, sent, withDatedLunches(menu, 2026, 10)["2026-10-01"])
}

func TestWithDatedLunches_NoWebWeeks_Unchanged(t *testing.T) {
	menu := map[string]any{"2026-10-05": map[string]any{"lunch": []any{"Çorba"}}}

	assert.Equal(t, menu, withDatedLunches(menu, 2026, 10))
}

func TestMenuService_SaveWebWeeks_MobileReadsTheDates(t *testing.T) {
	freezeAt(t, 2026, time.October, 5, 9, 0)
	svc := NewMenuService(&fakeMenus{rows: map[string]db.MonthlyMenu{}}, zap.NewNop())
	ctx := context.Background()

	_, err := svc.CreateOrUpdateMonthlyMenu(ctx, dto.CreateMonthlyMenuRequest{Year: 2026, Month: 10, MenuData: octoberWeeks()})
	require.NoError(t, err)

	saved, err := svc.GetMonthlyMenu(ctx, 2026, 10)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"lunch": []any{"Etli Nohut"}}, saved.MenuData["2026-10-12"])
}
