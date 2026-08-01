package auditlogs

type BaseAction struct {
	DisplayLabel string
	ActionIcon   string
	ActionURL    string
}

func (ba *BaseAction) Label() string {
	return ba.DisplayLabel
}

func (ba *BaseAction) Icon() string {
	return ba.ActionIcon
}

func (ba *BaseAction) URL() string {
	return ba.ActionURL
}
