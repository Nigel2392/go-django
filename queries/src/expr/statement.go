package expr

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"strconv"

	"github.com/Nigel2392/go-django/queries/internal"
)

const SELF_TABLE = "SELF" // the name of the self table, used in expressions

type statementInfoNode struct {
	start int
	end   int
	raw   string
	info  *statementInfo
}

var statementInfos = []*statementInfo{
	{
		typ:     "field",
		pattern: `!\[([^\]]*)\]`,
		rawtext: func(in []string) string {
			return in[1]
		},
		resolve: func(nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any) (string, []any, error) {
			var fieldName = in[1]
			var resolvedField = info.ResolveExpressionField(fieldName)
			return resolvedField.SQLText, resolvedField.SQLArgs, nil
		},
	},
	{
		typ:     "table",
		pattern: `#\[([^\]]*)\]`,
		rawtext: func(in []string) string {
			return in[1]
		},
		resolve: func(nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any) (string, []any, error) {
			var tableName = in[1]
			if tableName == SELF_TABLE {
				return info.Model.FieldDefs().TableName(), []any{}, nil
			}

			var current, _, _, _, aliases, _, err = internal.WalkFields(info.Model, tableName, info.AliasGen)
			if err != nil {
				panic(err)
			}

			if len(aliases) > 0 {
				return aliases[len(aliases)-1], []any{}, nil
			}

			var defs = current.FieldDefs()
			return info.QuoteIdentifier(defs.TableName()), []any{}, nil
		},
	},
	{
		typ:     "placeholder",
		pattern: `\?\[([^\]][0-9]*)\]|\?`,
		rawtext: func(in []string) string {
			return "?"
		},
		resolve: func(nodeIndex int, typIndex int, in []string, info *ExpressionInfo, args []any) (string, []any, error) {
			var valIdx = 0
			if len(in) > 1 && in[1] != "" {
				var err error
				valIdx, err = strconv.Atoi(in[1])
				if err != nil {
					panic(fmt.Errorf("invalid index %q in statement: %w", in[1], err))
				}
				valIdx-- // convert to 0-based index
			} else {
				valIdx = typIndex
			}
			if valIdx < 0 || valIdx >= len(args) {
				return "", nil, fmt.Errorf("index %d out of range in statement for %d arguments", valIdx, len(args))
			}
			var val = args[valIdx]
			return "?", []any{val}, nil
		},
	},
}

var (
	stmtBuilder = &statementBuilder{
		info: statementInfos,
	}
)

type statementInfo struct {
	typ     string
	pattern string
	rawtext func(in []string) string
	resolve func(index int, typIndex int, in []string, info *ExpressionInfo, args []any) (string, []any, error)

	_compiledAbs *regexp.Regexp
	_compiled    *regexp.Regexp // compiled regex, used for matching
}

func (inf *statementInfo) setup() {
	if inf._compiledAbs == nil {
		inf._compiledAbs = regexp.MustCompile(fmt.Sprintf(
			`^%s$`, inf.pattern,
		))
	}

	if inf._compiled == nil {
		inf._compiled = regexp.MustCompile(inf.pattern)
	}
}

type statementBuilder struct {
	info  []*statementInfo
	infos map[string]*statementInfo
}

type nodeResolver struct {
	stmt      string
	nodes     []statementInfoNode
	nodeTexts map[string][]string
}

func (r *nodeResolver) resolve(inf *ExpressionInfo, args []any) (string, []any, error) {
	var stmt bytes.Buffer
	var lastEnd = 0

	var argList = make([]any, 0, len(args))
	var seen = make(map[string]int, len(r.nodes))
	for nodeIdx, node := range r.nodes {
		var seenIdx = seen[node.info.typ]

		inStmt := r.stmt[node.start:node.end]
		match := node.info._compiledAbs.FindStringSubmatch(inStmt)
		if len(match) == 0 {
			return "", nil, fmt.Errorf("failed to match statement %q with pattern %q", inStmt, node.info.pattern)
		}

		var resolved, resolvedArgs, err = node.info.resolve(nodeIdx, seenIdx, match, inf, args)
		if err != nil {
			return "", nil, fmt.Errorf("failed to resolve node[%d.%d] %q: %w", nodeIdx, seenIdx, inStmt, err)
		}

		if resolved == "" {
			continue
		}

		argList = append(
			argList, resolvedArgs...,
		)

		stmt.Grow(node.start - lastEnd + len(resolved))
		stmt.WriteString(r.stmt[lastEnd:node.start])
		stmt.WriteString(resolved)
		lastEnd = node.end
		seen[node.info.typ] = seenIdx + 1
	}

	stmt.WriteString(r.stmt[lastEnd:]) // append remaining text
	return stmt.String(), argList, nil
}

func (b *statementBuilder) nodes(stmt string) *nodeResolver {

	var matches []statementInfoNode
	var stmtBytes = []byte(stmt)
	var nodeTexts = make(map[string][]string)
	for _, node := range b.info {
		node.setup()

		var m = node._compiled.FindAllStringSubmatchIndex(stmt, -1)
		if len(m) == 0 {
			continue
		}

		for _, match := range m {

			var (
				start = match[0]
				end   = match[1]
			)

			var rawBytes = stmtBytes[start:end]
			var absTextMatch = node._compiledAbs.FindStringSubmatch(string(rawBytes))
			if len(absTextMatch) == 0 {
				panic(fmt.Errorf("failed to match absolute pattern %q in statement %q", node.pattern, stmt))
			}

			var info = statementInfoNode{
				info:  node,
				start: start,
				end:   end,
				raw:   node.rawtext(absTextMatch),
			}

			var texts, ok = nodeTexts[node.typ]
			if !ok {
				texts = make([]string, 0, 1)
			}
			texts = append(texts, info.raw)
			nodeTexts[node.typ] = texts
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
