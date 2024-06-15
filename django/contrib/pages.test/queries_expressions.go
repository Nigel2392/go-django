package pages

type QueryExpression string

const (
	QueryExpressionEquals   QueryExpression = "equals"
	QueryExpressionIn       QueryExpression = "in"
	QueryExpressionNotIn    QueryExpression = "not_in"
	QueryExpressionStartsW  QueryExpression = "starts_with"
	QueryExpressionEndsW    QueryExpression = "ends_with"
	QueryExpressionContains QueryExpression = "contains"
)

type PageQueryExpression interface {
	Expression() QueryExpression
	Field() string
	Value() interface{}
}

type pageQueryExpressionImpl struct {
	expression QueryExpression
	field      string
	value      interface{}
}

func expression(expression QueryExpression, field string, value interface{}) PageQueryExpression {
	return &pageQueryExpressionImpl{
		expression: expression,
		field:      field,
		value:      value,
	}
}

func (p *pageQueryExpressionImpl) Expression() QueryExpression {
	return p.expression
}

func (p *pageQueryExpressionImpl) Field() string {
	return p.field
}

func (p *pageQueryExpressionImpl) Value() interface{} {
	return p.value
}
