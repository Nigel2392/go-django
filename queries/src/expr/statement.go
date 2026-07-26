package expr

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

const SELF_TABLE = "SELF" // the name of the self table, used in expressions

type StatementParser interface {
	// return the type / identifier of the statement parser, e.g. "field", "table", "value", "expr"
	Type() string

	// should return a value created by StatementParserArg of type parserArg
	Data(v any) any

	// Compiled returns the compiled regex for the statement parser
	Compiled() *regexp.Regexp

	// CompiledAbs returns the compiled regex for the statement parser with anchors
	CompiledAbs() *regexp.Regexp

	// RawText returns the raw text of the statement parser, given the matched input, I.E. ![FieldName] -> FieldName
	RawText(in []string) string

	// Resolve resolves the statement parser, given the matched input, the expression info, the arguments and any additional data
	Resolve(sb builder.Builder, nodesIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) error
}

type statement struct {
	Field      *statementParser
	ModelField *statementParser
	Table      *statementParser
	Value      *statementParser
	Quotes     *statementParser
	Expr       *expressionParser

	_map map[string]StatementParser // map of statement parsers by type
}

func (s *statement) Data(typ string, v any) any {
	if s._map == nil {
		s._map = make(map[string]StatementParser, 4)
		s._map[s.Field.Type()] = s.Field
		s._map[s.ModelField.Type()] = s.ModelField
		s._map[s.Quotes.Type()] = s.Quotes
		s._map[s.Table.Type()] = s.Table
		s._map[s.Value.Type()] = s.Value
		s._map[s.Expr.Type()] = s.Expr
	}

	var parser, ok = s._map[typ]
	if !ok {
		return nil
	}

	return parser.Data(v)
}

var PARSER = &statement{
	Field: &statementParser{
		typ:     "field",
		pattern: `\!\[([a-zA-Z][a-zA-Z0-9_.-]*)\]`, // ![FieldPath]
		rawtext: func(in []string) string {
			return in[1]
		},
		resolve: func(sb builder.Builder, nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) error {
			var fieldName = in[1]
			info.SupportsWhereAlias = false
			info.SupportsAsExpr = false

			var resolvedField = info.ResolveExpressionField(fieldName)
			sb.WriteString(resolvedField.SQLText)
			sb.AddVar(resolvedField.SQLArgs...)
			return nil
		},
	},
	Quotes: &statementParser{
		typ:     "quotes",
		pattern: `'([a-zA-Z][a-zA-Z0-9_.-]*)'`,
		rawtext: func(in []string) string {
			return in[1]
		},
		resolve: func(sb builder.Builder, nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) error {
			var flds = strings.Split(in[1], ".")
			for idx, field := range flds {
				if idx != 0 {
					sb.WriteRune('.')
				}
				sb.WriteString(info.QuoteIdentifier(field))
			}

			return nil
		},
	},
	ModelField: &statementParser{
		typ:     "modelfield",
		pattern: `\#\[(([a-z][a-z0-9]*\.|)([a-zA-Z][a-zA-Z0-9_]*)\.([A-Z][a-zA-Z0-9_]*)\.([a-zA-Z][a-zA-Z0-9_.]*))\]`, // #[auth.User.FieldPath] | #[t1.auth.User.FieldPath]
		rawtext: func(in []string) string {
			return in[1]
		},
		resolve: func(sb builder.Builder, nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) error {
			var (
				tableAlias = in[2]              // table alias
				modelPkg   = in[3]              // model pkg
				modelName  = in[4]              // model name
				fieldName  = tableAlias + in[5] // if table alias provided, it will be "<alias>.", so we can safely concat
			)

			contentType := contenttypes.DefinitionForPackage(modelPkg, modelName)
			if contentType == nil {
				return errors.NoTableName.Wrapf(
					"Could not find content type for %s.%s", modelPkg, modelName,
				)
			}

			info = info.CloneForModel(fmt.Sprintf("%s.%s", modelPkg, modelName), contentType.Object().(attrs.Definer))
			info.SupportsWhereAlias = false
			info.SupportsAsExpr = false
			resolvedField := info.ResolveExpressionField(fieldName)
			sb.WriteString(resolvedField.SQLText)
			sb.AddVar(resolvedField.SQLArgs...)
			return nil
		},
	},
	Value: &statementParser{
		typ:     "value",
		pattern: `(?:\?\[([0-9]+)\])|\?`, // ?[Index] or ?
		rawtext: func(in []string) string {
			return "?"
		},
		resolve: func(sb builder.Builder, nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) error {
			var valIdx = 0
			if len(in) > 1 && in[1] != "" {
				var err error
				valIdx, err = strconv.Atoi(in[1])
				if err != nil {
					return fmt.Errorf("invalid index %q in statement: %w", in[1], err)
				}

				if valIdx == 0 {
					return fmt.Errorf("invalid index %q in statement, use 1-based list indexing", in[1])
				}

				valIdx-- // convert to 0-based index
			} else {
				valIdx = typIndex
			}
			if valIdx < 0 || valIdx >= len(args) {
				return fmt.Errorf("index %d out of range in statement for %d arguments", valIdx, len(args))
			}
			var val = args[valIdx]

			if expr, ok := val.(Expression); ok {
				expr.Resolve(info).SQL(sb)
				return nil
			}

			sb.WriteRune('?')
			sb.AddVar(val)
			return nil
		},
	},
	Expr: &expressionParser{
		statementParser: statementParser{
			typ:     "expr",
			pattern: `(?:(?i)expr)\(((?:[a-zA-Z][a-zA-Z0-9_.-]*|[0-9]*))\)`, // expr(Index) or expr(ExpressionName)
			rawtext: func(in []string) string {
				return in[1]
			},
			resolve: func(sb builder.Builder, nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) error {
				if data == nil {
					return fmt.Errorf("expression data is nil for expr statement")
				}

				var exprData, ok = data.(*expressionData)
				if !ok {
					return fmt.Errorf("invalid expression data type for expr statement")
				}

				var (
					exprId = in[1]
					expr   Expression
				)

				// if the first number is a digit, we assume it's an index since
				// "<number><text>" is not valid according to the pattern
				if unicode.IsDigit(rune(exprId[0])) {
					var idx, err = strconv.Atoi(exprId)
					if err != nil {
						return fmt.Errorf("invalid expression index %q: %w", exprId, err)
					}
					if idx < 0 || idx >= len(exprData._list) {
						return fmt.Errorf("expression index %d out of range for %d expressions", idx, len(exprData._list))
					}

					expr = exprData._list[idx]

					// simpler, skips code duplication
					goto buildExpression
				}

				// assume the identifier is a name for an expression
				expr, ok = exprData._map[exprId]
				if !ok {
					return fmt.Errorf("expression %q not found in data", exprId)
				}

			buildExpression:
				expr.
					Resolve(info).
					SQL(sb)
				return nil
			},
		},
	},
	Table: &statementParser{
		typ: "table",
		// table(FieldPath), table(self), table(FieldPath as myCustomTable)
		pattern: `(?:[tT][aA][bB][lL][eE])\(([a-zA-Z][a-zA-Z0-9_.-]*)(?:|\s+[aA][sS]\s+(?:'|"|` + "`" + "|)([a-zA-Z][a-zA-Z0-9_-]*)(?:'|\"|" + "`" + `|))\)`,
		rawtext: func(in []string) string {
			return in[1]
		},
		resolve: func(sb builder.Builder, nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) error {
			var fieldPath = in[1]
			var asAlias = in[2]
			if strings.EqualFold(fieldPath, SELF_TABLE) {
				sb.WriteString(info.QuoteIdentifier(info.Resolver.Meta().TableName()))
				return nil
			}

			var _, field, _, err = info.Resolver.Resolve(fieldPath, info)
			if err != nil {
				var retErr = func(errs ...error) error {
					return fmt.Errorf(
						"error when walking fields: %w", errors.Join(errs...),
					)
				}

				if !errors.Is(err, errors.FieldNotFound) {
					return retErr(err)
				}

				var split = strings.Split(fieldPath, ".")
				if len(split) < 2 {
					return retErr(err)
				}

				contentType := contenttypes.DefinitionForPackage(split[0], split[1])
				if contentType == nil {
					return retErr(err)
				}

				var (
					ref  = contentType.Object()
					meta = attrs.GetModelMeta(ref)
					defs = meta.Definitions()
				)

				if len(split) == 2 {
					sb.WriteString(info.QuoteIdentifier(defs.TableName()))

					if asAlias != "" {
						sb.WriteString(" AS ")
						sb.WriteString(info.QuoteIdentifier(asAlias))
					}
					return nil
				}

				pathClone := strings.Join(split[2:], ".")
				_, field, _, err = info.
					CloneForModel(fmt.Sprintf("%s.%s", split[0], split[1]), ref.(attrs.Definer)).
					Resolver.Resolve(pathClone, info)
				fieldPath = pathClone
				if err != nil {
					return retErr(err)
				}
			}

			var rel = field.Rel()
			if rel == nil {
				return fmt.Errorf(
					"field %q is not a relation, cannot resolve table name", fieldPath,
				)
			}

			var (
				current        = rel.Model()
				defs           = attrs.Define(info.Resolver.Context(), current)
				tableName      = defs.TableName()
				lhs_tableName  = info.QuoteIdentifier(tableName)
				rhs_tableAlias string
			)

			if asAlias != "" {
				rhs_tableAlias = info.QuoteIdentifier(asAlias)
			} else {
				rhs_tableAlias = info.QuoteIdentifier(info.Resolver.Alias().GetTableAlias(
					tableName, fieldPath,
				))
			}

			sb.WriteString(lhs_tableName)
			sb.WriteString(" AS ")
			sb.WriteString(rhs_tableAlias)

			return nil
		},
	},
}

type statementParser struct {
	typ         string
	pattern     string
	processData func(data any) any // function to process the data, if needed
	rawtext     func(in []string) string
	resolve     func(sb builder.Builder, index int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) error

	_compiledAbs *regexp.Regexp
	_compiled    *regexp.Regexp // compiled regex, used for matching
}

type expressionData struct {
	_map  map[string]Expression
	_list []Expression
}

type expressionParser struct {
	statementParser
}

func parseExpressionsFromArgs(expr ...any) (list []Expression, _map map[string]Expression) {
	var (
		exprs_list = make([]Expression, 0, len(expr))
		exprs_map  = make(map[string]Expression, len(expr))
		manualMap  bool
		mapKey     string
	)

	for i := 0; i < len(expr); i++ {
		e := expr[i]

		switch {
		case i == 0 || (i%2 == 0 && manualMap):
			mapKey, manualMap = e.(string)
			if manualMap {
				continue
			}

		case manualMap:
			var ex Expression
			ex, manualMap = e.(Expression)
			if manualMap {
				exprs_map[mapKey] = ex
				continue
			}
		}

		switch v := e.(type) {
		case NamedExpression:
			var name = v.FieldName()
			if name == "" {
				exprs_list = append(exprs_list, v)
			} else {
				exprs_map[name] = v
			}
		case Expression:
			exprs_list = append(exprs_list, v)
		case []any:
			var _list, _map = parseExpressionsFromArgs(v...)
			exprs_list = append(exprs_list, _list...)
			maps.Copy(exprs_map, _map)
		case map[string]any:
			for key, value := range v {
				var _list, _map = parseExpressionsFromArgs(value)
				for _, expr := range _list {
					exprs_map[key] = expr
				}
				maps.Copy(exprs_map, _map)
			}
		case map[string]Expression:
			maps.Copy(exprs_map, v)
		default:
			panic(fmt.Errorf("invalid expression type %T, expected Expression or NamedExpression", e))
		}
	}

	return exprs_list, exprs_map
}
func (inf *expressionParser) Data(expr any) any {
	return inf.Expressions(expr)
}

func (inf *expressionParser) Expressions(expr ...any) any {
	var exprs_list, exprs_map = parseExpressionsFromArgs(expr...)
	return inf.statementParser.Data(&expressionData{
		_map:  exprs_map,
		_list: exprs_list,
	})
}

func (inf *statementParser) Data(v any) any {
	if inf.processData != nil {
		v = inf.processData(v)
	}

	return StatementParserArg(inf.typ, v)
}

func (inf *statementParser) Type() string {
	return inf.typ
}

func (inf *statementParser) Compiled() *regexp.Regexp {
	if inf._compiled == nil {
		inf._compiled = regexp.MustCompile(inf.pattern)
	}

	return inf._compiled
}

func (inf *statementParser) CompiledAbs() *regexp.Regexp {
	if inf._compiledAbs == nil {
		inf._compiledAbs = regexp.MustCompile(fmt.Sprintf(
			`^%s$`, inf.pattern,
		))
	}

	return inf._compiledAbs
}

func (inf *statementParser) RawText(in []string) string {
	if inf.rawtext == nil {
		panic(fmt.Errorf("rawtext function not defined for statement type %q", inf.typ))
	}
	return inf.rawtext(in)
}

func (inf *statementParser) Resolve(sb builder.Builder, nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) error {
	if inf.resolve == nil {
		panic(fmt.Errorf("resolve function not defined for statement type %q", inf.typ))
	}
	return inf.resolve(sb, nodeIndex, typIndex, in, info, args, data)
}

type statementBuilder struct {
	info []StatementParser
}

type statementInfoNode struct {
	start int
	end   int
	raw   string
	info  StatementParser
}

type parserArg struct {
	which string // the type of the node
	data  any
}

// StatementParserArg creates a parserArg for the given type and data.
//
// a parserArg is a special case used in [ExpressionStatement.Resolve] to
// pass additional data to a registered parser.
func StatementParserArg(which string, data any) any {
	if which == "" {
		panic(fmt.Errorf("NodeArg must have a non-empty 'which' field"))
	}
	if data == nil {
		panic(fmt.Errorf("NodeArg must have a non-nil 'data' field"))
	}
	return parserArg{
		which: which,
		data:  data,
	}
}

type nodeResolver struct {
	stmt      string
	nodes     []statementInfoNode
	nodeTexts map[string][]string
}

func (r *nodeResolver) resolve(inf *ExpressionInfo, args []any) (string, []any, error) {
	var params = make([]any, 0, len(args))
	var data = make(map[string]any, len(r.nodes))
	for _, arg := range args {
		switch v := arg.(type) {
		case parserArg:
			if v.which == "" {
				return "", nil, fmt.Errorf("parserArg must have a non-empty 'which' field")
			}
			if v.data == nil {
				return "", nil, fmt.Errorf("parserArg must have a non-nil 'data' field")
			}
			data[v.which] = v.data
		default:
			params = append(params, v)
		}
	}

	var lastEnd = 0
	var builder = builder.BaseBuilder{}
	var seen = make(map[string]int, len(r.nodes))
	for nodeIdx, node := range r.nodes {
		var nodeType = node.info.Type()
		var seenIdx = seen[nodeType]
		var inStmt = r.stmt[node.start:node.end]
		var pattern = node.info.CompiledAbs()
		var match = pattern.FindStringSubmatch(inStmt)
		if len(match) == 0 {
			return "", nil, fmt.Errorf("failed to match statement %q with pattern %q", inStmt, pattern.String())
		}

		builder.Grow(node.start - lastEnd)
		builder.WriteString(r.stmt[lastEnd:node.start])

		var err = node.info.Resolve(&builder, nodeIdx, seenIdx, match, inf, params, data[nodeType])
		if err != nil {
			return "", nil, fmt.Errorf("failed to resolve node[%d.%d] %q: %w", nodeIdx, seenIdx, inStmt, err)
		}

		lastEnd = node.end
		seen[nodeType] = seenIdx + 1
	}

	builder.WriteString(r.stmt[lastEnd:]) // append remaining text
	return builder.String(), builder.Vars, builder.GetError()
}

func (b *statementBuilder) nodes(stmt string) *nodeResolver {

	var matches []statementInfoNode
	var stmtBytes = []byte(stmt)
	var nodeTexts = make(map[string][]string)
	for _, node := range b.info {

		var m = node.Compiled().FindAllStringSubmatchIndex(stmt, -1)
		if len(m) == 0 {
			continue
		}

		for _, match := range m {

			var (
				nodeType = node.Type()
				start    = match[0]
				end      = match[1]
			)

			var rawBytes = stmtBytes[start:end]
			var pattern = node.CompiledAbs()
			var absTextMatch = pattern.FindStringSubmatch(string(rawBytes))
			if len(absTextMatch) == 0 {
				panic(fmt.Errorf("failed to match absolute pattern %q in statement %q", pattern.String(), stmt))
			}

			var info = statementInfoNode{
				info:  node,
				start: start,
				end:   end,
				raw:   node.RawText(absTextMatch),
			}

			var texts, ok = nodeTexts[nodeType]
			if !ok {
				texts = make([]string, 0, 1)
			}
			texts = append(texts, info.raw)
			nodeTexts[nodeType] = texts
			matches = append(matches, info)
		}
	}

	slices.SortFunc(matches, func(a, b statementInfoNode) int {
		if a.start < b.start {
			return -1
		} else if a.start > b.start {
			return 1
		}
		return 0
	})

	return &nodeResolver{
		stmt:      stmt,
		nodes:     matches,
		nodeTexts: nodeTexts,
	}
}

var stmtBuilder = &statementBuilder{
	info: []StatementParser{
		PARSER.Field,
		PARSER.ModelField,
		PARSER.Quotes,
		PARSER.Table,
		PARSER.Value,
		PARSER.Expr,
	},
}

type expressionStatementInfo struct {
	used         bool
	resolver     *nodeResolver
	resolvedSQL  string
	resolvedArgs []any
}

type ExpressionStatement struct {
	info      expressionStatementInfo
	Statement string
	Values    []any
}

func (s *ExpressionStatement) Raw(which string) []string {
	if s.info.resolver == nil {
		s.info.resolver = stmtBuilder.nodes(s.Statement)
	}

	var rawTexts, ok = s.info.resolver.nodeTexts[which]
	if !ok {
		return []string{}
	}

	return slices.Clone(rawTexts)
}

func (s *ExpressionStatement) Clone() *ExpressionStatement {
	if s.info.resolver == nil {
		s.info.resolver = stmtBuilder.nodes(s.Statement)
	}
	return &ExpressionStatement{
		info: expressionStatementInfo{
			used:         s.info.used,
			resolvedSQL:  s.info.resolvedSQL,
			resolvedArgs: slices.Clone(s.info.resolvedArgs),
			resolver: &nodeResolver{
				stmt:  s.info.resolver.stmt,
				nodes: slices.Clone(s.info.resolver.nodes),
			},
		},
		Statement: s.Statement,
		Values:    slices.Clone(s.Values),
	}
}

func (s *ExpressionStatement) Resolve(inf *ExpressionInfo) *ExpressionStatement {
	if s.info.used {
		return s
	}

	s = s.Clone()

	var err error
	s.info.used = true
	s.info.resolvedSQL, s.info.resolvedArgs, err = s.info.resolver.resolve(inf, s.Values)
	if err != nil {
		panic(fmt.Errorf("failed to resolve statement %q: %w", s.Statement, err))
	}

	return s
}

func (s *ExpressionStatement) SQL() (string, []any) {
	if !s.info.used {
		panic("statement not resolved, call Resolve first")
	}

	return s.info.resolvedSQL, s.info.resolvedArgs
}

// The statement should contain placeholders for the fields and values, which will be replaced with the actual values.
//
// The placeholders for fields should be in the format ![FieldName], and the placeholders for values should be in the format ?[Index],
// or the values should use the regular SQL placeholder directly (database driver dependent).
//
// Example usage:
//
//	 stmt := ParseExprStatement(
//			"SELECT * FROM TABLE(SELF as p) WHERE ![p.Field1] = ?[0] AND ![p.Field2] = ?[1] AND EXPR(MyExpression)",
//			"users", 42, "active",
//	     expr.PARSER.Expressions(map[string]expr.Expression{
//				"MyExpression": expr.Q("p.Field2", "MyTitle")
//			})
//		)
func ParseExprStatement(statement string, value []any) *ExpressionStatement {

	return &ExpressionStatement{
		Statement: statement,
		info: expressionStatementInfo{
			used:     false,
			resolver: stmtBuilder.nodes(statement),
		},
		Values: value,
	}
}
