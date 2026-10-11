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

func TestWithDinnersKept_WebSave_KeepsStoredDinners(t *testing.T) {
	stored := map[string]any{
		"normalMenus": []any{},
		"2026-10-12":  map[string]any{"lunch": []any{"Eski Pilav"}, "dinner": []any{"Köfte"}},
		"2026-10-13":  map[string]any{"dinner": []any{"Balık"}},
		"2026-10-14":  map[string]any{"lunch": []any{"Çorba"}},
	}
	menu := withDatedLunches(octoberWeeks(), 2026, 10)

	kept := withDinnersKept(menu, stored)

	assert.Equal(t, map[string]any{"lunch": []any{"Etli Nohut"}, "dinner": []any{"Köfte"}}, kept["2026-10-12"])
	assert.Equal(t, map[string]any{"dinner": []any{"Balık"}}, kept["2026-10-13"], "a dinner outlives a day the web leaves empty")
	assert.NotContains(t, kept, "2026-10-14", "a stored lunch is the save's to replace")
	assert.NotContains(t, menu["2026-10-12"], "dinner", "the save itself is not changed")
}

func TestWithDinnersKept_DinnerInSave_SaveWins(t *testing.T) {
	stored := map[string]any{"2026-10-12": map[string]any{"dinner": []any{"Köfte"}}}
	menu := map[string]any{"2026-10-12": map[string]any{"dinner": []any{"Mantı"}}}

	assert.Equal(t, menu["2026-10-12"], withDinnersKept(menu, stored)["2026-10-12"])
}

func TestMenuService_WebEditAfterSeed_MobileKeepsDinner(t *testing.T) {
	freezeAt(t, 2026, time.October, 5, 9, 0)
	svc := NewMenuService(&fakeMenus{rows: map[string]db.MonthlyMenu{}}, zap.NewNop())
	ctx := context.Background()
	seeded := octoberWeeks()
	seeded["2026-10-12"] = map[string]any{"lunch": []any{"Etli Nohut"}, "dinner": []any{"Köfte"}}
	_, err := svc.CreateOrUpdateMonthlyMenu(ctx, dto.CreateMonthlyMenuRequest{Year: 2026, Month: 10, MenuData: seeded})
	require.NoError(t, err)

	edited := octoberWeeks()
	edited["normalMenus"].([]any)[2] = map[string]any{"monday": webDay("Kuru Fasulye")}
	_, err = svc.CreateOrUpdateMonthlyMenu(ctx, dto.CreateMonthlyMenuRequest{Year: 2026, Month: 10, MenuData: edited})
	require.NoError(t, err)

	saved, err := svc.GetMonthlyMenu(ctx, 2026, 10)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"lunch": []any{"Kuru Fasulye"}, "dinner": []any{"Köfte"}}, saved.MenuData["2026-10-12"])
}

func TestMenuService_DateFormatSave_ReplacesDinners(t *testing.T) {
	freezeAt(t, 2026, time.October, 5, 9, 0)
	svc := NewMenuService(&fakeMenus{rows: map[string]db.MonthlyMenu{}}, zap.NewNop())
	ctx := context.Background()
	for _, day := range []map[string]any{
		{"lunch": []any{"Çorba"}, "dinner": []any{"Köfte"}},
		{"lunch": []any{"Çorba"}},
	} {
		_, err := svc.CreateOrUpdateMonthlyMenu(ctx, dto.CreateMonthlyMenuRequest{Year: 2026, Month: 10, MenuData: map[string]any{"2026-10-12": day}})
		require.NoError(t, err)
	}

	saved, err := svc.GetMonthlyMenu(ctx, 2026, 10)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"lunch": []any{"Çorba"}}, saved.MenuData["2026-10-12"], "a client that speaks dates owns the dinner too")
}
