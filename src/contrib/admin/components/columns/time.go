package columns

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"time"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/views/list"
)

type TimeInformation struct {
	Source  time.Time
	Years   int
	Months  int
	Days    int
	Hours   int
	Minutes int
	Seconds int
	Future  bool
}

func GetTimeDiffInformation(t time.Time, diffFrom time.Time) (info TimeInformation) {
	other := diffFrom.In(t.Location())
	if t.After(other) {
		t, other = other, t // swap, so we always diff "earlier to later"
		info.Future = true
	}

	temp := t

	// Years
	for temp.AddDate(1, 0, 0).Before(other) || temp.AddDate(1, 0, 0).Equal(other) {
		temp = temp.AddDate(1, 0, 0)
		info.Years++
	}

	// Months
	for temp.AddDate(0, 1, 0).Before(other) || temp.AddDate(0, 1, 0).Equal(other) {
		temp = temp.AddDate(0, 1, 0)
		info.Months++
	}

	// Days
	for temp.AddDate(0, 0, 1).Before(other) || temp.AddDate(0, 0, 1).Equal(other) {
		temp = temp.AddDate(0, 0, 1)
		info.Days++
	}

	duration := other.Sub(temp)
	info.Source = t
	info.Hours = int(duration.Hours())
	info.Minutes = int(duration.Minutes()) % 60
	info.Seconds = int(duration.Seconds()) % 60
	return info
}

// FormatTimeAgo formats the time difference nicely, e.g. "1 year, 2 months, 3 weeks ago"
func FormatTimeDifference(ctx context.Context, t time.Time, diffFrom time.Time) ([]string, TimeInformation) {
	var parts = []string{}
	var info = GetTimeDiffInformation(t, diffFrom)
	if info.Years > 0 {
		parts = append(parts, trans.P(ctx, "%d year", "%d years", info.Years, info.Years))
	}
	if info.Months > 0 {
		parts = append(parts, trans.P(ctx, "%d month", "%d months", info.Months, info.Months))
	}
	if info.Days >= 7 {
		weeks := info.Days / 7
		parts = append(parts, trans.P(ctx, "%d week", "%d weeks", weeks, weeks))
		info.Days = info.Days % 7
	}
	if info.Days > 0 && len(parts) < 2 {
		parts = append(parts, trans.P(ctx, "%d day", "%d days", info.Days, info.Days))
	}
	if info.Hours > 0 && len(parts) == 0 {
		parts = append(parts, trans.P(ctx, "%d hour", "%d hours", info.Hours, info.Hours))
	}
	if info.Minutes > 0 && len(parts) == 0 {
		parts = append(parts, trans.P(ctx, "%d minute", "%d minutes", info.Minutes, info.Minutes))
	}

	return parts, info
}

func FormatTimeSince(ctx context.Context, t time.Time, diffFrom time.Time) string {
	var parts, info = FormatTimeDifference(ctx, t, diffFrom)
	// Only show up to two largest units, e.g. "1 year, 2 months ago"
	if len(parts) > 2 {
		parts = parts[:2]
	}

	var text string
	if info.Future {
		switch len(parts) {
		case 0:
		case 1:
			text = trans.T(ctx, "in %s", parts[0])
		default:
			text = trans.T(ctx, "in %s and %s", parts[0], parts[1])
		}
	} else {
		switch len(parts) {
		case 0:
			text = trans.T(ctx, "just now")
		case 1:
			text = trans.T(ctx, "%s ago", parts[0])
		default:
			text = trans.T(ctx, "%s and %s ago", parts[0], parts[1])
		}
	}

	return text
}

func TimeSinceColumn[T attrs.Definer](label any, field string, hoverFormat ...string) list.ListColumn[T] {
	var timeFormat = trans.LONG_TIME_FORMAT
	if len(hoverFormat) > 0 {
		timeFormat = hoverFormat[0]
	}

	return list.HTMLColumn(
		trans.GetTextFunc(label),
		func(r *http.Request, defs attrs.Definitions, row T) template.HTML {
			var timeField, ok = defs.Field(field)
			assert.True(
				ok, "Field %q not found in definitions for %T",
				field, row,
			)

			var timeFace, err = timeField.Value()
			assert.True(
				err == nil, "Failed to get value for field %q in %T: %v",
				field, row, err,
			)

			var timeRval = reflect.ValueOf(timeFace)
			timeRval = timeRval.Convert(reflect.TypeOf(time.Time{}))
			timeVal := timeRval.Interface().(time.Time)

			if timeVal.IsZero() {
				return template.HTML(fmt.Sprintf(
					`<span class="badge warning">%s</span>`,
					trans.T(r.Context(), "Never"),
				))
			}

			return template.HTML(fmt.Sprintf(
				`<span class="badge" data-controller="tooltip" data-tooltip-content-value="%s" data-tooltip-placement-value="%s" data-tooltip-offset-value="[0, %v]">%s</span>`,
				trans.Time(r.Context(), timeVal, timeFormat), "bottom-start", 10, FormatTimeSince(
					r.Context(), timeVal, time.Now(),
				),
			))
		},
	)
}
