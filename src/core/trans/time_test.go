package trans_test

import (
	"context"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/core/trans"
)

var DEFAULT_TIME, _ = time.Parse(
	"2006-01-02 15:04:05",
	"2023-10-01 12:53:23",
)

type timeFormatTest struct {
	Format   string
	Expected string
}

var timeFormatTests = []timeFormatTest{
	{
		Format:   "%Y-%m-%d %H:%M:%S",
		Expected: "2023-10-01 12:53:23",
	},
	{
		Format:   "%d/%m/%Y",
		Expected: "01/10/2023",
	},
	{
		Format:   "%I:%M %p",
		Expected: "12:53 PM",
	},
	{
		Format:   "%A, %B %d, %Y",
		Expected: "Sunday, October 01, 2023",
	},
	{
		Format:   "%Y-%m-%d %H:%M:%S %z",
		Expected: "2023-10-01 12:53:23 +00:00",
	},
	{
		Format:   "%Y-%m-%d %H:%M:%S %Z",
		Expected: "2023-10-01 12:53:23 UTC",
	},
	{
		Format:   "%Y-%m-%d %H:%M:%S %Z %z",
		Expected: "2023-10-01 12:53:23 UTC +00:00",
	},
	{
		Format:   "%Y",
		Expected: "2023",
	},
	{
		Format:   "%m",
		Expected: "10",
	},
	{
		Format:   "%d",
		Expected: "01",
	},
	{
		Format:   "It is %A, %B %d, %Y at %I:%M %p",
		Expected: "It is Sunday, October 01, 2023 at 12:53 PM",
	},
	{
		// Test escaping of percent sign
		Format:   "Current time 100%%: %H:%M:%S",
		Expected: "Current time 100%: 12:53:23",
	},
	{
		// Test escaping of percent sign with multiple percent signs
		Format:   "Current time 100%%%Y-%m-%d %%: %H:%M:%S",
		Expected: "Current time 100%2023-10-01 %: 12:53:23",
	},
	{
		// Test week of year
		Format:   "%U",
		Expected: "39", // 1st week of October 2023
	},

	{
		Format:   "%-m",
		Expected: "10",
	},
	{
		Format:   "%-d",
		Expected: "1",
	},
	{
		Format:   "%-y",
		Expected: "23",
	},
	{
		Format:   "%-Y",
		Expected: "23",
	},
	{
		Format:   "%-H",
		Expected: "12",
	},
	{
		Format:   "%-I",
		Expected: "12",
	},
	{
		Format:   "%-M",
		Expected: "53",
	},
	{
		Format:   "%-S",
		Expected: "23",
	},
	{
		Format:   "%-f",
		Expected: "0",
	},
	{
		Format:   "%-F",
		Expected: "0",
	},
}

func TestTimeFormat(t *testing.T) {
	for _, test := range timeFormatTests {
		t.Run(test.Format, func(t *testing.T) {
			result := trans.Time(context.Background(), DEFAULT_TIME, test.Format)
			if result != test.Expected {
				t.Errorf("expected %s, got %s", test.Expected, result)
			}
		})
	}
}
