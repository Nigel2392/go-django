package pages

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/mux"
)

var (
	_ mux.Resolver = &pageRouteResolver{}
	_ mux.Handler  = &pageRouteResolver{}
)

type pageRouteResolver struct {
	mux.Handler
}

// This is a custom resolver for page routes that allows us to reverse the URL of a page
// based on its ID or Page object. It also matches the URL to the page's slug.
//
// It can be used with django.Reverse to generate URLs for pages, example:
//
//	django.Reverse("pages:page", page)
//	django.Reverse("pages:page", page.ID())
//
// When passed anything other than a Page, string, int64 or value which can be converted to int64,
// it will return an error.
func (p *pageRouteResolver) Reverse(baseURL string, variables ...interface{}) (string, error) {
	if len(variables) == 0 {
		return "", mux.ErrNotEnoughVariables
	}
	if len(variables) > 1 {
		return "", mux.ErrTooManyVariables
	}

	var pageID int64
	switch v := variables[0].(type) {
	case Page:
		pageID = v.ID()
	case int64:
		pageID = v
	case string:
		var id, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return "", errors.ValueError.WithCause(err).Wrapf(
				"cannot reverse page route, expected a Page or number type, got %T", variables[0],
			)
		}
		pageID = id
	default:
		var lhs = reflect.ValueOf(pageID)
		var rhs = reflect.ValueOf(variables[0])

		if !rhs.IsValid() {
			return "", errors.ValueError.Wrapf(
				"cannot reverse page route, expected a Page or number type, got %T", variables[0],
			)
		}

		if lhs.Type() != rhs.Type() && !rhs.Type().ConvertibleTo(lhs.Type()) {
			return "", errors.TypeMismatch
		}

		if lhs.Type() != rhs.Type() {
			rhs = rhs.Convert(lhs.Type())
		}

		if !rhs.IsValid() {
			return "", errors.ValueError.Wrapf(
				"cannot reverse page route, expected a Page or number type, got %T", variables[0],
			)
		}

		pageID = rhs.Int()
	}

	var query = `WITH RECURSIVE parent_walk AS (
    -- Start with the node of interest
    SELECT
        ![node.Path],
        ![node.Slug],
        0 AS level
    FROM TABLE(SELF) node
    WHERE ![node.PK] = ?

    UNION ALL
	SELECT
		![p.Path],
		![p.Slug],
		pw.level + 1 AS level
    FROM TABLE(SELF) p
    INNER JOIN parent_walk pw
      ON ![p.Path] = EXPR(pwPathSubstr)
)
SELECT *
FROM parent_walk
ORDER BY level DESC;`

	var rows, err = queries.GetQuerySet(&PageNode{}).Rows(
		query,
		expr.PARSER.Expr.Expressions(map[string]expr.Expression{
			"pwPathSubstr": expr.SUBSTR(
				"pw.Path", 1,
				expr.LENGTH("pw.Path").SUB(expr.Value(STEP_LEN))),
		}),

		pageID,
		STEP_LEN,
	)
	if err != nil {
		return "", err
	}

	defer rows.Close()

	var urlParts []string
	for rows.Next() {
		var (
			path  string
			level int64
			slug  string
		)

		if err := rows.Scan(&path, &slug, &level); err != nil {
			return "", err
		}

		if level == 0 {
			// Skip the root node
			continue
		}

		urlParts = append(urlParts, slug)
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"%s%s",
		baseURL,
		strings.Join(urlParts, "/"),
	), nil

	//siteRows, err := queries.GetQuerySet(&Site{}).
	//	Filter("Root", rootID).
	//	OrderBy("Default").
	//	All()
	//if err != nil {
	//	return "", err
	//}
	//
	//if len(siteRows) == 0 {
	//	return fmt.Sprintf("%s%s", baseURL, strings.Join(urlParts, "/")), nil
	//	//	return "", errors.NoRows.Wrapf(
	//	//		"no site found for page with ID %d", pageID,
	//	//	)
	//}
	//
	//var site = siteRows[0].Object
	//return fmt.Sprintf(
	//	"%s%s%s",
	//	site.URL(),
	//	baseURL,
	//	strings.Join(urlParts, "/"),
	//), nil
}

func (p *pageRouteResolver) Match(vars mux.Variables, path []string) (bool, mux.Variables) {
	vars[mux.GLOB] = path
	return true, vars
}
