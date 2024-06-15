package pages

type QueryType string

const (
	QueryTypeAnd QueryType = "and"
	QueryTypeOr  QueryType = "or"
)

type PageQuery interface {
	Type() QueryType
	Expressions() []PageQueryExpression
	Action() PageQueryAction
}

func Query(t QueryType, action PageQueryAction, expressions ...PageQueryExpression) PageQuery {
	return &pageQueryImpl{
		queryType:   t,
		expressions: expressions,
		action:      action,
	}
}

type pageQueryImpl struct {
	queryType   QueryType
	expressions []PageQueryExpression
	action      PageQueryAction
}

func (p *pageQueryImpl) Type() QueryType {
	return p.queryType
}

func (p *pageQueryImpl) Expressions() []PageQueryExpression {
	return p.expressions
}

func (p *pageQueryImpl) Action() PageQueryAction {
	return p.action
}
