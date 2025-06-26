package expr

import (
	"bytes"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/Nigel2392/go-django/queries/internal"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

const SELF_TABLE = "SELF" // the name of the self table, used in expressions

type StatementParser interface {
	Type() string
	Data(v any) any // should return an parserArg with the type of the statement and the data
	Compiled() *regexp.Regexp
	CompiledAbs() *regexp.Regexp
	RawText(in []string) string
	Resolve(nodesIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) (string, []any, error)
}

type statement struct {
	Field *statementParser
	Table *statementParser
	Value *statementParser
	Expr  *expressionParser

	_map map[string]StatementParser // map of statement parsers by type
}

func (s *statement) Data(typ string, v any) any {
	if s._map == nil {
		s._map = make(map[string]StatementParser, 4)
		s._map[s.Field.Type()] = s.Field
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

var STMT = &statement{
	Field: &statementParser{
		typ:     "field",
		pattern: `\!\[([a-zA-Z][a-zA-Z0-9_.-]*)\]`, // ![FieldPath]
		rawtext: func(in []string) string {
			return in[1]
		},
		resolve: func(nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) (string, []any, error) {
			var fieldName = in[1]
			var resolvedField = info.ResolveExpressionField(fieldName)
			return resolvedField.SQLText, resolvedField.SQLArgs, nil
		},
	},
	Table: &statementParser{
		typ:     "table",
		pattern: `(?:(?i)table)\(([a-zA-Z][a-zA-Z0-9_.-]*)\)`, // table(FieldPath)
		rawtext: func(in []string) string {
			return in[1]
		},
		resolve: func(nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) (string, []any, error) {
			var fieldPath = in[1]
			if fieldPath == SELF_TABLE {
				var meta = attrs.GetModelMeta(info.Model)
				var defs = meta.Definitions()
				return info.QuoteIdentifier(defs.TableName()), []any{}, nil
			}

			var current, _, _, _, aliases, _, err = internal.WalkFields(info.Model, fieldPath, info.AliasGen)
			if err != nil {
				return "", []any{}, fmt.Errorf(
					"error when walking fields: %w", err,
				)
			}

			var tableName string
			if len(aliases) > 0 {
				tableName = aliases[len(aliases)-1]
			} else {
				var defs = current.FieldDefs()
				tableName = defs.TableName()
			}

			return info.QuoteIdentifier(tableName), []any{}, nil
		},
	},
	Value: &statementParser{
		typ:     "value",
		pattern: `(?:\?\[([0-9]+)\])|\?`, // ?[Index] or ?
		rawtext: func(in []string) string {
			return "?"
		},
		resolve: func(nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) (string, []any, error) {
			var valIdx = 0
			if len(in) > 1 && in[1] != "" {
				var err error
				valIdx, err = strconv.Atoi(in[1])
				if err != nil {
					return "", []any{}, fmt.Errorf("invalid index %q in statement: %w", in[1], err)
				}
				valIdx-- // convert to 0-based index
			} else {
				valIdx = typIndex
			}
			if valIdx < 0 || valIdx >= len(args) {
				return "", nil, fmt.Errorf("index %d out of range in statement for %d arguments", valIdx, len(args))
			}
			var val = args[valIdx]

			if expr, ok := val.(Expression); ok {
				var exprStr strings.Builder
				var exprParams = expr.Resolve(info).SQL(&exprStr)
				return exprStr.String(), exprParams, nil
			}

			return "?", []any{val}, nil
		},
	},
	Expr: &expressionParser{
		statementParser: statementParser{
			typ:     "expr",
			pattern: `(?:(?i)expr)\(((?:[a-zA-Z][a-zA-Z0-9_.-]*|[0-9]*))\)`, // expr(Index) or expr(ExpressionName)
			rawtext: func(in []string) string {
				return in[1]
			},
			resolve: func(nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) (string, []any, error) {
				if data == nil {
					return "", nil, fmt.Errorf("expression data is nil for expr statement")
				}

				var exprData, ok = data.(*expressionData)
				if !ok {
					return "", nil, fmt.Errorf("invalid expression data type for expr statement")
				}

				var (
					exprId = in[1]
					expr   Expression
				)

				if unicode.IsDigit(rune(exprId[0])) {
					var idx, err = strconv.Atoi(exprId)
					if err != nil {
						return "", nil, fmt.Errorf("invalid expression index %q: %w", exprId, err)
					}
					if idx < 0 || idx >= len(exprData._list) {
						return "", nil, fmt.Errorf("expression index %d out of range for %d expressions", idx, len(exprData._list))
					}
					expr = exprData._list[idx]
					goto buildExpression
				}

				expr, ok = exprData._map[exprId]
				if !ok {
					return "", nil, fmt.Errorf("expression %q not found in data", exprId)
				}

			buildExpression:
				expr = expr.Resolve(info)
				var exprStr strings.Builder
				var exprParams = expr.SQL(&exprStr)
				return exprStr.String(), exprParams, nil
			},
		},
	},
}

var (
	stmtBuilder = &statementBuilder{
		info: []StatementParser{
			STMT.Field,
			STMT.Table,
			STMT.Value,
			STMT.Expr,
		},
	}
)

type statementParser struct {
	typ         string
	pattern     string
	processData func(data any) any // function to process the data, if needed
	rawtext     func(in []string) string
	resolve     func(index int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) (string, []any, error)

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
	)

	for _, e := range expr {
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

	return ParserArg(inf.typ, v)
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

func (inf *statementParser) Resolve(nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any, data any) (string, []any, error) {
	if inf.resolve == nil {
		panic(fmt.Errorf("resolve function not defined for statement type %q", inf.typ))
	}
	return inf.resolve(nodeIndex, typIndex, in, info, args, data)
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

func ParserArg(which string, data any) any {
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

	var stmt bytes.Buffer
	var lastEnd = 0
	var argList = make([]any, 0, len(params))
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

		var resolved, resolvedArgs, err = node.info.Resolve(nodeIdx, seenIdx, match, inf, params, data[nodeType])
		if err != nil {
			return "", nil, fmt.Errorf("failed to resolve node[%d.%d] %q: %w", nodeIdx, seenIdx, inStmt, err)
		}

		argList = append(
			argList, resolvedArgs...,
		)

		stmt.Grow(node.start - lastEnd + len(resolved))
		stmt.WriteString(r.stmt[lastEnd:node.start])
		stmt.WriteString(resolved)
		lastEnd = node.end
		seen[nodeType] = seenIdx + 1
	}

	stmt.WriteString(r.stmt[lastEnd:]) // append remaining text
	return stmt.String(), argList, nil
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
//	 # sets the field name to the first field found in the statement, I.E. ![Field1]:
//
//		stmt, fields, values := ParseExprStatement("![Field1] = ![Age] + ?[1] + ![Height] + ?[2] * ?[1]", 3, 4)
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
