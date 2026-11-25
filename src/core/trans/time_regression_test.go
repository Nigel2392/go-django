package trans_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/core/trans"
	"golang.org/x/text/language"
)

func TestTimeClockBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		hour int
		want string
	}{
		{"midnight", 0, "00/0 12/12 05/5 07/7 AM"},
		{"early morning", 1, "01/1 01/1 05/5 07/7 AM"},
		{"last morning hour", 11, "11/11 11/11 05/5 07/7 AM"},
		{"noon", 12, "12/12 12/12 05/5 07/7 PM"},
		{"early afternoon", 13, "13/13 01/1 05/5 07/7 PM"},
		{"last hour", 23, "23/23 11/11 05/5 07/7 PM"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := time.Date(2024, time.January, 1, tc.hour, 5, 7, 0, time.UTC)
			const format = "%H/%-H %I/%-I %M/%-M %S/%-S %p"
			if got := trans.Time(t.Context(), value, format); got != tc.want {
				t.Errorf("Time = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTimeTimezoneOffsets(t *testing.T) {
	for _, tc := range []struct {
		name   string
		offset time.Duration
		want   string
	}{
		{"UTC", 0, "12:00 UTC +00:00"},
		{"JST", 9 * time.Hour, "21:00 JST +09:00"},
		{"PST", -8 * time.Hour, "04:00 PST -08:00"},
		{"IST", 5*time.Hour + 30*time.Minute, "17:30 IST +05:30"},
		{"NPT", 5*time.Hour + 45*time.Minute, "17:45 NPT +05:45"},
		{"NST", -3*time.Hour - 30*time.Minute, "08:30 NST -03:30"},
		{"West", -30 * time.Minute, "11:30 West -00:30"},
		{"East", 30 * time.Minute, "12:30 East +00:30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC)
			value := start.In(time.FixedZone(tc.name, int(tc.offset.Seconds())))
			if got := trans.Time(t.Context(), value, "%H:%M %Z %z"); got != tc.want {
				t.Errorf("Time = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTimeCalendarBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name  string
		year  int
		month time.Month
		day   int
		want  string
	}{
		{"Gregorian leap century", 2000, time.February, 29, "2000/00 2000/00 02/2 29/29 060"},
		{"non-leap century", 1900, time.March, 1, "1900/00 1900/00 03/3 01/1 060"},
		{"day after leap day", 2024, time.March, 1, "2024/24 2024/24 03/3 01/1 061"},
		{"end of leap year", 2024, time.December, 31, "2024/24 2024/24 12/12 31/31 366"},
		{"end of common year", 2023, time.December, 31, "2023/23 2023/23 12/12 31/31 365"},
		{"first day of year one", 1, time.January, 1, "0001/01 0001/01 01/1 01/1 001"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := time.Date(tc.year, tc.month, tc.day, 0, 0, 0, 0, time.UTC)
			const format = "%Y/%-Y %y/%-y %m/%-m %d/%-d %j"
			if got := trans.Time(t.Context(), value, format); got != tc.want {
				t.Errorf("Time = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTimeWeekdays(t *testing.T) {
	monday := time.Date(2024, time.December, 30, 0, 0, 0, 0, time.UTC)
	for i, want := range []string{"Mon Monday 1", "Tue Tuesday 2", "Wed Wednesday 3", "Thu Thursday 4", "Fri Friday 5", "Sat Saturday 6", "Sun Sunday 7"} {
		t.Run(want, func(t *testing.T) {
			value := monday.AddDate(0, 0, i)
			if got := trans.Time(t.Context(), value, "%a %A %w"); got != want {
				t.Errorf("Time = %q, want %q", got, want)
			}
		})
	}
}

func TestTimeMonths(t *testing.T) {
	for i, want := range []string{
		"Jan January", "Feb February", "Mar March", "Apr April", "May May", "Jun June",
		"Jul July", "Aug August", "Sep September", "Oct October", "Nov November", "Dec December",
	} {
		t.Run(want, func(t *testing.T) {
			value := time.Date(2024, time.Month(i+1), 1, 0, 0, 0, 0, time.UTC)
			if got := trans.Time(t.Context(), value, "%b %B"); got != want {
				t.Errorf("Time = %q, want %q", got, want)
			}
		})
	}
}

func TestTimeFractionalSeconds(t *testing.T) {
	for _, tc := range []struct {
		nanos int
		want  string
	}{
		{0, "000/0 000000/0"},
		{999, "000/0 000000/0"},
		{1_000, "000/0 000001/1"},
		{999_999, "000/0 000999/999"},
		{1_000_000, "001/1 001000/1000"},
		{123_456_789, "123/123 123456/123456"},
		{999_999_999, "999/999 999999/999999"},
	} {
		t.Run(fmt.Sprint(tc.nanos), func(t *testing.T) {
			value := time.Date(2024, time.January, 1, 0, 0, 59, tc.nanos, time.UTC)
			if got := trans.Time(t.Context(), value, "%f/%-f %F/%-F"); got != tc.want {
				t.Errorf("Time = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTimeISOWeekBoundaries(t *testing.T) {
	for _, tc := range []struct {
		year  int
		month time.Month
		day   int
		want  string
	}{
		{2019, time.December, 30, "01/01"},
		{2020, time.December, 31, "53/53"},
		{2021, time.January, 1, "53/53"},
		{2021, time.January, 3, "53/53"},
		{2021, time.January, 4, "01/01"},
	} {
		value := time.Date(tc.year, tc.month, tc.day, 0, 0, 0, 0, time.UTC)
		t.Run(value.Format(time.DateOnly), func(t *testing.T) {
			if got := trans.Time(t.Context(), value, "%U/%W"); got != tc.want {
				t.Errorf("Time = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTimeFormatLiterals(t *testing.T) {
	for _, tc := range []struct {
		name, format, want string
	}{
		{"empty", "", ""},
		{"literal only", "時刻 🕰️", "時刻 🕰️"},
		{"escaped sequence", "%%Y%%m%%d", "%Y%m%d"},
		{"literal between fields", "%Y年%m月%d日", "2024年02月09日"},
		{"escaped percent before specifier", "%%%d", "%09"},
		{"flag applies to only one field", "%-d%d%-m%m", "909202"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := time.Date(2024, time.February, 9, 0, 0, 0, 0, time.UTC)
			if got := trans.Time(t.Context(), value, tc.format); got != tc.want {
				t.Errorf("Time(%q) = %q, want %q", tc.format, got, tc.want)
			}
		})
	}
}

func TestTimeRejectsInvalidSpecifiers(t *testing.T) {
	for _, tc := range []struct {
		name, format, want string
	}{
		{"unknown specifier", "%Q", "unknown format specifier Q"},
		{"unsupported flag", "%-A", "format specifier A does not support flag -"},
		{"recognized flag without a supporting directive", "%!d", "format specifier d does not support flag !"},
		{"duplicate flag", "%--m", "duplicate flag -"},
		{"escape after a flag", "date %-%", "unexpected %"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if got := recover(); got == nil || !strings.Contains(fmt.Sprint(got), tc.want) {
					t.Errorf("Time(%q) panic = %v, want %q", tc.format, got, tc.want)
				}
			}()
			trans.Time(t.Context(), time.Time{}, tc.format)
		})
	}
}

type frenchTimeBackend struct{ trans.SprintBackend }

func (frenchTimeBackend) Translate(ctx context.Context, value string) string {
	if trans.LocaleFromContext(ctx) == language.French {
		if translated, ok := map[string]string{
			"Thursday": "jeudi", "Thu": "jeu.", "February": "février", "Feb": "févr.", "PM": "après-midi",
		}[value]; ok {
			return translated
		}
	}
	return value
}

func (frenchTimeBackend) TimeFormat(ctx context.Context, short bool) string {
	if trans.LocaleFromContext(ctx) != language.French {
		return trans.DEFAULT_TIME_FORMAT
	}
	if short {
		return "%a %-d %b"
	}
	return "%A %-d %B %Y, %I:%M %p"
}

func TestTimeUsesBackendFormatsAndLocale(t *testing.T) {
	prev := trans.DefaultBackend
	t.Cleanup(func() { trans.DefaultBackend = prev })
	trans.DefaultBackend = &frenchTimeBackend{}
	value := time.Date(2024, time.February, 29, 13, 5, 0, 0, time.UTC)

	for _, tc := range []struct {
		name, format, want string
		locale             language.Tag
	}{
		{"short format", trans.SHORT_TIME_FORMAT, "jeu. 29 févr.", language.French},
		{"long format", trans.LONG_TIME_FORMAT, "jeudi 29 février 2024, 01:05 après-midi", language.French},
		{"explicit localized directives", "%a / %B / %p", "jeu. / février / après-midi", language.French},
		{"explicit source directives", "%a / %B / %p", "Thu / February / PM", language.English},
		{"different locale has different default", trans.SHORT_TIME_FORMAT, "2024-02-29 13:05:00", language.English},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := trans.ContextWithLocale(t.Context(), tc.locale)
			if got := trans.Time(ctx, value, tc.format); got != tc.want {
				t.Errorf("Time(%q) = %q, want %q", tc.format, got, tc.want)
			}
		})
	}
}
