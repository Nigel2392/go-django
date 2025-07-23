package pages

import (
	"reflect"
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

func (p *pageRouteResolver) Reverse(variables ...interface{}) (string, error) {
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
		urlParts = append(urlParts, slug)
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	return strings.Join(urlParts, "/"), nil
}

func (p *pageRouteResolver) Match(vars mux.Variables, path []string) (bool, mux.Variables) {
	vars[mux.GLOB] = path
	return true, vars
}
