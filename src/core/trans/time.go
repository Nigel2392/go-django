package trans

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

var (
	Monday    = S("Monday")
	Tuesday   = S("Tuesday")
	Wednesday = S("Wednesday")
	Thursday  = S("Thursday")
	Friday    = S("Friday")
	Saturday  = S("Saturday")
	Sunday    = S("Sunday")

	ShortMonday    = S("Mon")
	ShortTuesday   = S("Tue")
	ShortWednesday = S("Wed")
	ShortThursday  = S("Thu")
	ShortFriday    = S("Fri")
	ShortSaturday  = S("Sat")
	ShortSunday    = S("Sun")

	January   = S("January")
	February  = S("February")
	March     = S("March")
	April     = S("April")
	May       = S("May")
	June      = S("June")
	July      = S("July")
	August    = S("August")
	September = S("September")
	October   = S("October")
	November  = S("November")
	December  = S("December")

	ShortJanuary   = S("Jan")
	ShortFebruary  = S("Feb")
	ShortMarch     = S("Mar")
	ShortApril     = S("Apr")
	ShortMay       = S("May")
	ShortJune      = S("Jun")
	ShortJuly      = S("Jul")
	ShortAugust    = S("Aug")
	ShortSeptember = S("Sep")
	ShortOctober   = S("Oct")
	ShortNovember  = S("Nov")
	ShortDecember  = S("Dec")

	AM   = S("AM")
	PM   = S("PM")
	AMPM = S("AM/PM")
)

type timeInfo struct {
	time   time.Time
	year   int
	month  time.Month
	week   time.Weekday
	day    int
	hour   int
	minute int
	second int
	millis int
	micros int
	nanos  int
}

func newTimeInfo(t time.Time) *timeInfo {
	return &timeInfo{
		time:   t,
		year:   t.Year(),
		month:  t.Month(),
		week:   t.Weekday(),
		day:    t.Day(),
		hour:   t.Hour(),
		minute: t.Minute(),
		second: t.Second(),
		millis: t.Nanosecond() / 1e6,
		micros: t.Nanosecond() / 1e3,
		nanos:  t.Nanosecond(),
	}
}

var (
	formatMap = map[string]func(ctx context.Context, t *timeInfo) Translation{
		"a": func(ctx context.Context, t *timeInfo) Translation { // short weekday name
			var day = t.week
			switch day {
			case time.Monday:
				return ShortMonday(ctx)
			case time.Tuesday:
				return ShortTuesday(ctx)
			case time.Wednesday:
				return ShortWednesday(ctx)
			case time.Thursday:
				return ShortThursday(ctx)
			case time.Friday:
				return ShortFriday(ctx)
			case time.Saturday:
				return ShortSaturday(ctx)
			case time.Sunday:
				return ShortSunday(ctx)
			default:
				return day.String()
			}
		},
		"A": func(ctx context.Context, t *timeInfo) Translation { // full weekday name
			var day = t.week
			var text string
			switch day {
			case time.Monday:
				text = Monday(ctx)
			case time.Tuesday:
				text = Tuesday(ctx)
			case time.Wednesday:
				text = Wednesday(ctx)
			case time.Thursday:
				text = Thursday(ctx)
			case time.Friday:
				text = Friday(ctx)
			case time.Saturday:
				text = Saturday(ctx)
			case time.Sunday:
				text = Sunday(ctx)
			default:
				text = day.String()
			}

			return text
		},
		"w": func(ctx context.Context, t *timeInfo) Translation { // weekday number (1-7, Monday is 1, Sunday is 7)
			var day = t.week
			switch day {
			case time.Monday:
				return "1"
			case time.Tuesday:
				return "2"
			case time.Wednesday:
				return "3"
			case time.Thursday:
				return "4"
			case time.Friday:
				return "5"
			case time.Saturday:
				return "6"
			case time.Sunday:
				return "7"
			default:
				return day.String()
			}
		},
		"b": func(ctx context.Context, t *timeInfo) Translation { // full month name
			var month = t.month
			switch month {
			case time.January:
				return ShortJanuary(ctx)
			case time.February:
				return ShortFebruary(ctx)
			case time.March:
				return ShortMarch(ctx)
			case time.April:
				return ShortApril(ctx)
			case time.May:
				return ShortMay(ctx)
			case time.June:
				return ShortJune(ctx)
			case time.July:
				return ShortJuly(ctx)
			case time.August:
				return ShortAugust(ctx)
			case time.September:
				return ShortSeptember(ctx)
			case time.October:
				return ShortOctober(ctx)
			case time.November:
				return ShortNovember(ctx)
			case time.December:
				return ShortDecember(ctx)
			default:
				return month.String()
			}
		},
		"B": func(ctx context.Context, t *timeInfo) Translation { // full month name
			var month = t.month
			switch month {
			case time.January:
				return January(ctx)
			case time.February:
				return February(ctx)
			case time.March:
				return March(ctx)
			case time.April:
				return April(ctx)
			case time.May:
				return May(ctx)
			case time.June:
				return June(ctx)
			case time.July:
				return July(ctx)
			case time.August:
				return August(ctx)
			case time.September:
				return September(ctx)
			case time.October:
				return October(ctx)
			case time.November:
				return November(ctx)
			case time.December:
				return December(ctx)
			default:
				return month.String()
			}
		},
		"m": func(ctx context.Context, t *timeInfo) Translation { // month number (01-12)
			var month = t.month
			return fmt.Sprintf("%02d", month)
		},
		"-m": func(ctx context.Context, t *timeInfo) Translation { // month number (1-12)
			var month = t.month
			return strconv.Itoa(int(month))
		},
		"d": func(ctx context.Context, t *timeInfo) Translation { // day of the month (01-31)
			var day = t.day
			return fmt.Sprintf("%02d", day)
		},
		"-d": func(ctx context.Context, t *timeInfo) Translation { // day
			var day = t.day
			return strconv.Itoa(day)
		},
		"y": func(ctx context.Context, t *timeInfo) Translation { // year (4 digits)
			var year = t.year
			return fmt.Sprintf("%04d", year)
		},
		"-y": func(ctx context.Context, t *timeInfo) Translation { // year (2 digits)
			var year = t.year
			return fmt.Sprintf("%02d", year%100)
		},
		"Y": func(ctx context.Context, t *timeInfo) Translation { // year (4 digits)
			var year = t.year
			return fmt.Sprintf("%04d", year)
		},
		"-Y": func(ctx context.Context, t *timeInfo) Translation { // year (2 digits)
			var year = t.year
			return fmt.Sprintf("%02d", year%100)
		},
		"H": func(ctx context.Context, t *timeInfo) Translation { // hour (00-23)
			var hour = t.hour
			return fmt.Sprintf("%02d", hour)
		},
		"-H": func(ctx context.Context, t *timeInfo) Translation { // hour (0-23)
			var hour = t.hour
			return strconv.Itoa(hour)
		},
		"I": func(ctx context.Context, t *timeInfo) Translation { // hour (01-12)
			var hour = t.hour
			if hour == 0 {
				hour = 12
			}
			if hour > 12 {
				hour -= 12
			}
			return fmt.Sprintf("%02d", hour)
		},
		"-I": func(ctx context.Context, t *timeInfo) Translation { // hour (1-12)
			var hour = t.hour
			if hour == 0 {
				hour = 12
			}
			if hour > 12 {
				hour -= 12
			}
			return strconv.Itoa(hour)
		},
		"M": func(ctx context.Context, t *timeInfo) Translation { // minute (
			var minute = t.minute
			return fmt.Sprintf("%02d", minute)
		},
		"-M": func(ctx context.Context, t *timeInfo) Translation { // minute (0-59)
			var minute = t.minute
			return strconv.Itoa(minute)
		},
		"S": func(ctx context.Context, t *timeInfo) Translation { // second (00-59)
			var second = t.second
			return fmt.Sprintf("%02d", second)
		},
		"-S": func(ctx context.Context, t *timeInfo) Translation { // second (0-59)
			var second = t.second
			return strconv.Itoa(second)
		},
		"f": func(ctx context.Context, t *timeInfo) Translation { // milliseconds (000-999)
			var millis = t.millis
			return fmt.Sprintf("%03d", millis)
		},
		"-f": func(ctx context.Context, t *timeInfo) Translation { // milliseconds (0-999)
			var millis = t.millis
			return strconv.Itoa(millis)
		},
		"F": func(ctx context.Context, t *timeInfo) Translation { // microseconds (000000-999999)
			var micros = t.micros
			return fmt.Sprintf("%06d", micros)
		},
		"-F": func(ctx context.Context, t *timeInfo) Translation { // microseconds (0-999999)
			var micros = t.micros
			return strconv.Itoa(micros)
		},
		"p": func(ctx context.Context, t *timeInfo) Translation { // AM/PM
			var hour = t.hour
			if hour < 12 {
				return AM(ctx)
			}
			if hour == 12 {
				return PM(ctx)
			}
			if hour > 12 {
				return PM(ctx)
			}
			return AM(ctx)
		},
		"z": func(ctx context.Context, t *timeInfo) Translation { // timezone offset
			_, offset := t.time.Zone()
			return fmt.Sprintf("%+03d:00", offset/3600)
		},
		"Z": func(ctx context.Context, t *timeInfo) Translation { // timezone name
			name, _ := t.time.Zone()
			return name
		},
		"j": func(ctx context.Context, t *timeInfo) Translation { // day of the year (001-366)
			dayOfYear := t.time.YearDay()
			return fmt.Sprintf("%03d", dayOfYear)
		},
		"U": func(ctx context.Context, t *timeInfo) Translation { // week number (00-53, Sunday as first day of week)
			_, week := t.time.ISOWeek()
			return fmt.Sprintf("%02d", week)
		},
		"W": func(ctx context.Context, t *timeInfo) Translation { // week number (00-53, Monday as first day of week)
			_, week := t.time.ISOWeek()
			return fmt.Sprintf("%02d", week)
		},
	}
)
