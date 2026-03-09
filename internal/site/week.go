package site

import (
	"fmt"
	"time"
)

// BiWeeklyPeriod holds the current bi-weekly period info for content creation.
type BiWeeklyPeriod struct {
	Week      int    // even ISO week number
	Month     string // "March"
	MonthNum  string // "03"
	Year      string // "2026"
	YearShort string // "26"
	Filename  string // "10-03-26.md"
	Dir       string // "2026/March" (relative to content/)
}

// CurrentBiWeeklyPeriod returns the bi-weekly period for the current date.
func CurrentBiWeeklyPeriod() BiWeeklyPeriod {
	return BiWeeklyPeriodFor(time.Now())
}

// BiWeeklyPeriodFor returns the bi-weekly period for a given date.
func BiWeeklyPeriodFor(t time.Time) BiWeeklyPeriod {
	_, isoWeek := t.ISOWeek()

	// Round down to even week (bi-weekly periods start on even weeks)
	evenWeek := isoWeek
	if evenWeek%2 != 0 {
		evenWeek--
	}

	monthNum := fmt.Sprintf("%02d", t.Month())
	yearShort := t.Format("06")
	year := t.Format("2006")
	month := t.Format("January")

	return BiWeeklyPeriod{
		Week:      evenWeek,
		Month:     month,
		MonthNum:  monthNum,
		Year:      year,
		YearShort: yearShort,
		Filename:  fmt.Sprintf("%02d-%s-%s.md", evenWeek, monthNum, yearShort),
		Dir:       fmt.Sprintf("%s/%s", year, month),
	}
}
