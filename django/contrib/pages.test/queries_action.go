package pages

type QueryAction string

const (
	QueryActionSelect QueryAction = "select"
	QueryActionCreate QueryAction = "create"
	QueryActionUpdate QueryAction = "update"
	QueryActionDelete QueryAction = "delete"
)

type PageQueryAction interface {
	Action() QueryAction
	Fields() []string
	Values() []interface{}
}

type pageQueryActionImpl struct {
	action QueryAction
	fields []string
	values []interface{}
}

func action(action QueryAction, fields []string, values []interface{}) PageQueryAction {
	return &pageQueryActionImpl{
		action: action,
		fields: fields,
		values: values,
	}
}

func (p *pageQueryActionImpl) Action() QueryAction {
	return p.action
}

func (p *pageQueryActionImpl) Fields() []string {
	return p.fields
}

func (p *pageQueryActionImpl) Values() []interface{} {
	return p.values
}
