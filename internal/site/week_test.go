package site

import (
	"testing"
	"time"
)

func TestBiWeeklyPeriodEvenWeek(t *testing.T) {
	// 2026-03-09 is ISO week 11 → rounds to 10
	d := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	p := BiWeeklyPeriodFor(d)

	if p.Week != 10 {
		t.Errorf("Week = %d, want 10", p.Week)
	}
	if p.Month != "March" {
		t.Errorf("Month = %q, want March", p.Month)
	}
	if p.MonthNum != "03" {
		t.Errorf("MonthNum = %q, want 03", p.MonthNum)
	}
	if p.Year != "2026" {
		t.Errorf("Year = %q, want 2026", p.Year)
	}
	if p.YearShort != "26" {
		t.Errorf("YearShort = %q, want 26", p.YearShort)
	}
	if p.Filename != "10-03-26.md" {
		t.Errorf("Filename = %q, want 10-03-26.md", p.Filename)
	}
	if p.Dir != "2026/March" {
		t.Errorf("Dir = %q, want 2026/March", p.Dir)
	}
}

func TestBiWeeklyPeriodAlreadyEven(t *testing.T) {
	// 2026-03-02 is ISO week 10 (even) → stays 10
	d := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	p := BiWeeklyPeriodFor(d)

	if p.Week != 10 {
		t.Errorf("Week = %d, want 10", p.Week)
	}
}

func TestBiWeeklyPeriodJanuary(t *testing.T) {
	// 2026-01-05 is ISO week 2 (even) → stays 2
	d := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	p := BiWeeklyPeriodFor(d)

	if p.Week != 2 {
		t.Errorf("Week = %d, want 2", p.Week)
	}
	if p.Month != "January" {
		t.Errorf("Month = %q, want January", p.Month)
	}
	if p.Filename != "02-01-26.md" {
		t.Errorf("Filename = %q, want 02-01-26.md", p.Filename)
	}
}

func TestBiWeeklyPeriodWeekOne(t *testing.T) {
	// ISO week 1 (odd) → rounds to 0, which should clamp to 0
	// 2026-01-01 is ISO week 1
	d := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	p := BiWeeklyPeriodFor(d)

	// Week 1 rounds down to 0
	if p.Week != 0 {
		t.Errorf("Week = %d, want 0", p.Week)
	}
}

func TestBiWeeklyPeriodDecember(t *testing.T) {
	// 2026-12-28 is ISO week 53 or 1 depending on year
	d := time.Date(2026, 12, 28, 12, 0, 0, 0, time.UTC)
	p := BiWeeklyPeriodFor(d)

	if p.Month != "December" {
		t.Errorf("Month = %q, want December", p.Month)
	}
	if p.Year != "2026" {
		t.Errorf("Year = %q, want 2026", p.Year)
	}
	// Week should be even
	if p.Week%2 != 0 {
		t.Errorf("Week %d is not even", p.Week)
	}
}

func TestBiWeeklyPeriodDifferentYears(t *testing.T) {
	dates := []time.Time{
		time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 11, 20, 0, 0, 0, 0, time.UTC),
	}

	for _, d := range dates {
		p := BiWeeklyPeriodFor(d)

		// Week should always be even
		if p.Week%2 != 0 {
			t.Errorf("date %s: Week %d is not even", d.Format("2006-01-02"), p.Week)
		}

		// Filename should match pattern WW-MM-YY.md
		if len(p.Filename) < 10 {
			t.Errorf("date %s: Filename %q too short", d.Format("2006-01-02"), p.Filename)
		}

		// Dir should be YYYY/MonthName
		if p.Dir == "" {
			t.Errorf("date %s: Dir is empty", d.Format("2006-01-02"))
		}
	}
}
