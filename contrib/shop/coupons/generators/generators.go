package generators

import (
	"context"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/google/uuid"
)

type CodePartType uint

const (
	CPT_RANDOM  CodePartType = iota // generate a random string as the code part
	CPT_STATIC                      // use a static string as code part
	CPT_YEAR                        // use current year as code part
	CPT_MONTH                       // use current month as code part
	CPT_WEEKDAY                     // use current weekday as code part
	CPT_DAY                         // use current day as code part
	CPT_HOUR                        // use current hour as code part
	CPT_MINUTE                      // use current minute as code part
	CPT_DASH                        // use a dash for the code part
	CPT_UUID                        // use random uuid as code part

	ErrInvalidPartType    errs.Error = "invalid code part type"
	ErrCouponCodeTooShort errs.Error = "coupon code is too short"
	ErrCouponCodeTooLong  errs.Error = "coupon code is too long"
	ErrCouponExists       errs.Error = "coupon already exists"
)

var partStrings = map[CodePartType]string{
	CPT_RANDOM:  "CPT_RANDOM",
	CPT_STATIC:  "CPT_STATIC",
	CPT_YEAR:    "CPT_YEAR",
	CPT_MONTH:   "CPT_MONTH",
	CPT_WEEKDAY: "CPT_WEEKDAY",
	CPT_DAY:     "CPT_DAY",
	CPT_HOUR:    "CPT_HOUR",
	CPT_MINUTE:  "CPT_MINUTE",
	CPT_DASH:    "CPT_DASH",
	CPT_UUID:    "CPT_UUID",
}

func (c CodePartType) String() string {
	s, ok := partStrings[c]
	if !ok {
		return "CPT_UNKNOWN"
	}
	return s
}

type CodePart struct {
	// type of the code part
	Type CodePartType `json:"type"`

	// length of the code part
	// is only used in case of random or uuid code parts
	Length int `json:"length"`

	// value of the code part
	// is only used in case of static code parts
	Value string `json:"value"`
}

const letterBytes = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func generateCouponCode(parts []CodePart) (string, bool, error) {
	var (
		sb strings.Builder
		t  = time.Now()
	)

	var isStatic bool = true
	for _, part := range parts {
		var val string
		switch part.Type {
		case CPT_RANDOM:
			sb.WriteString(randStringBytes(part.Length))
			isStatic = false
			continue
		case CPT_STATIC:
			sb.WriteString(part.Value)
			continue
		case CPT_YEAR:
			sb.WriteString(strconv.Itoa(t.Year()))
			continue
		case CPT_MONTH:
			sb.WriteString(strconv.Itoa(int(t.Month())))
		case CPT_WEEKDAY:
			sb.WriteString(strconv.Itoa(int(t.Weekday())))
			continue
		case CPT_DAY:
			sb.WriteString(strconv.Itoa(t.Day()))
			continue
		case CPT_HOUR:
			sb.WriteString(strconv.Itoa(t.Hour()))
			continue
		case CPT_MINUTE:
			sb.WriteString(strconv.Itoa(t.Minute()))
			continue
		case CPT_DASH:
			sb.WriteRune('-')
			continue
		case CPT_UUID:
			isStatic = false
			u := uuid.New()
			val = strings.Replace(u.String(), "-", "", -1)
		default:
			return "", false, ErrInvalidPartType
		}

		if part.Length > 0 && part.Length < len(val) {
			val = val[:part.Length]
		}

		sb.WriteString(val)
	}

	if sb.Len() < int(coupons.MIN_CODE_LENGTH) {
		return "", false, ErrCouponCodeTooLong
	}

	if sb.Len() > int(coupons.MAX_CODE_LENGTH) {
		return "", false, ErrCouponCodeTooLong
	}

	return sb.String(), isStatic, nil
}

func GenerateCouponCode(ctx context.Context, parts []CodePart) (code string, err error) {
	var qs = queries.GetQuerySetWithContext(ctx, &coupons.Coupon{})
	var isStatic bool
	for {
		code, isStatic, err = generateCouponCode(parts)
		if err != nil {
			return "", errors.Wrap(err, "GenerateCouponCode")
		}

		exists, err := qs.
			Filter("Code", code).
			Exists()
		if err != nil {
			return "", errors.Wrap(err, "GenerateCouponCode")
		}

		if !exists {
			return code, nil
		}

		if isStatic {
			return "", errors.Wrap(err, "GenerateCouponCode")
		}
	}
}
