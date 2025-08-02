package trans

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

const (
	DEFAULT_TIME_FORMAT = "%Y-%m-%d %H:%M:%S"
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

type timeFormatter struct {
	supportedFlags []byte
	format         func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation

	_flagsMap map[byte]bool
}

func (f *timeFormatter) supportsFlag(flag byte) bool {
	if len(f._flagsMap) != len(f.supportedFlags) {
		f._flagsMap = make(map[byte]bool, len(f.supportedFlags))
		for _, supportedFlag := range f.supportedFlags {
			f._flagsMap[supportedFlag] = true
		}
	}
	_, ok := f._flagsMap[flag]
	return ok
}

var (
	formatMap = map[byte]timeFormatter{
		'a': {
			supportedFlags: []byte{},
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation {
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
		},
		'A': {
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // full weekday name
				var day = t.week
				switch day {
				case time.Monday:
					return Monday(ctx)
				case time.Tuesday:
					return Tuesday(ctx)
				case time.Wednesday:
					return Wednesday(ctx)
				case time.Thursday:
					return Thursday(ctx)
				case time.Friday:
					return Friday(ctx)
				case time.Saturday:
					return Saturday(ctx)
				case time.Sunday:
					return Sunday(ctx)
				}
				return day.String()
			},
		},
		'w': {
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // weekday number (1-7, Monday is 1, Sunday is 7)
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
		},
		'b': {
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // full month name
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
		},
		'B': {
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // full month name
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
		},
		'm': {
			supportedFlags: []byte{'-'},
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // month number (01-12)
				var month = t.month
				if _, ok := flags['-']; ok {
					return strconv.Itoa(int(month))
				}
				return fmt.Sprintf("%02d", month)
			},
		},
		'd': {
			supportedFlags: []byte{'-'},
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // day of the month (01-31)
				var day = t.day
				if _, ok := flags['-']; ok {
					return strconv.Itoa(day)
				}
				return fmt.Sprintf("%02d", day)
			},
		},
		'y': {
			supportedFlags: []byte{'-'},
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // year (4 digits)
				var year = t.year
				if _, ok := flags['-']; ok {
					return fmt.Sprintf("%02d", year%100)
				}
				return fmt.Sprintf("%04d", year)
			},
		},
		'Y': {
			supportedFlags: []byte{'-'},
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // year (4 digits)
				var year = t.year
				if _, ok := flags['-']; ok {
					return fmt.Sprintf("%02d", year%100)
				}
				return fmt.Sprintf("%04d", year)
			},
		},
		'H': {
			supportedFlags: []byte{'-'},
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // hour (00-23)
				var hour = t.hour
				if _, ok := flags['-']; ok {
					return fmt.Sprintf("%02d", hour)
				}
				return strconv.Itoa(hour)
			},
		},
		'I': {
			supportedFlags: []byte{'-'},
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // hour (01-12)
				var hour = t.hour
				if hour == 0 {
					hour = 12
				}
				if hour > 12 {
					hour -= 12
				}
				if _, ok := flags['-']; ok {
					return strconv.Itoa(hour)
				}
				return fmt.Sprintf("%02d", hour)
			},
		},
		'M': {
			supportedFlags: []byte{'-'},
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // minute (00-59)
				var minute = t.minute
				if _, ok := flags['-']; ok {
					return strconv.Itoa(minute)
				}
				return fmt.Sprintf("%02d", minute)
			},
		},
		'S': {
			supportedFlags: []byte{'-'},
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // second (00-59)
				var second = t.second
				if _, ok := flags['-']; ok {
					return strconv.Itoa(second)
				}
				return fmt.Sprintf("%02d", second)
			},
		},
		'f': {
			supportedFlags: []byte{'-'},
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // milliseconds (000-999)
				var millis = t.millis
				if _, ok := flags['-']; ok {
					return strconv.Itoa(millis)
				}
				return fmt.Sprintf("%03d", millis)
			},
		},
		'F': {
			supportedFlags: []byte{'-'},
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // microseconds (000000-999999)
				var micros = t.micros
				if _, ok := flags['-']; ok {
					return strconv.Itoa(micros)
				}
				return fmt.Sprintf("%06d", micros)
			},
		},
		'p': {
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // AM/PM
				if t.hour == 12 {
					return PM(ctx)
				}
				if t.hour > 12 {
					return PM(ctx)
				}
				return AM(ctx)
			},
		},
		'z': {
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // timezone offset
				_, offset := t.time.Zone()
				return fmt.Sprintf("%+03d:00", offset/3600)
			},
		},
		'Z': {
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // timezone name
				name, _ := t.time.Zone()
				return name
			},
		},
		'j': {
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // day of the year (001-366)
				dayOfYear := t.time.YearDay()
				return fmt.Sprintf("%03d", dayOfYear)
			},
		},
		'U': {
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // week number (00-53, Sunday as first day of week)
				_, week := t.time.ISOWeek()
				return fmt.Sprintf("%02d", week)
			},
		},
		'W': {
			format: func(ctx context.Context, t *timeInfo, flags map[byte]bool) Translation { // week number (00-53, Monday as first day of week)
				_, week := t.time.ISOWeek()
				return fmt.Sprintf("%02d", week)
			},
		},
	}
)
