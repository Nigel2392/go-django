package checks

type CheckResult interface {
	Messages() []Message
	Debugs() []Message
	Infos() []Message
	Warnings() []Message
	Errors() []Message
	Criticals() []Message
}

type checksResult struct {
	m      []Message
	byType map[Type][]Message
}

func (r *checksResult) filterByType(typ Type) []Message {
	if r.byType == nil {
		r.byType = make(map[Type][]Message)
	}

	if messages, ok := r.byType[typ]; ok {
		return messages
	}

	var filtered []Message
	for _, msg := range r.m {
		if msg.Type == typ && !msg.Silenced() {
			filtered = append(filtered, msg)
		}
	}

	r.byType[typ] = filtered
	return filtered
}

func (r *checksResult) Messages() []Message {
	return r.m
}

func (r *checksResult) Debugs() []Message {
	return r.filterByType(DEBUG)
}

func (r *checksResult) Infos() []Message {
	return r.filterByType(INFO)
}

func (r *checksResult) Warnings() []Message {
	return r.filterByType(WARNING)
}

func (r *checksResult) Errors() []Message {
	return r.filterByType(ERROR)
}

func (r *checksResult) Criticals() []Message {
	return r.filterByType(CRITICAL)
}
