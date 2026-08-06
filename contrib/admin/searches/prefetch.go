package searches

type Prefetch struct {
	SelectRelated   []string
	PrefetchRelated []any
}

func (p *Prefetch) Merge(other Prefetch) {
	var (
		thisSelectMap   = make(map[string]struct{})
		thisPrefetchMap = make(map[any]struct{})
	)

	for _, s := range p.SelectRelated {
		thisSelectMap[s] = struct{}{}
	}
	for _, s := range p.PrefetchRelated {
		thisPrefetchMap[s] = struct{}{}
	}

	var selectRelated = make([]string, len(p.SelectRelated), len(thisSelectMap)+len(other.SelectRelated))
	var prefetchRelated = make([]any, len(p.PrefetchRelated), len(thisPrefetchMap)+len(other.PrefetchRelated))
	copy(selectRelated, p.SelectRelated)
	copy(prefetchRelated, p.PrefetchRelated)

	for _, s := range other.SelectRelated {
		if _, ok := thisSelectMap[s]; !ok {
			selectRelated = append(selectRelated, s)
		}
	}

	for _, s := range other.PrefetchRelated {
		if _, ok := thisPrefetchMap[s]; !ok {
			prefetchRelated = append(prefetchRelated, s)
		}
	}

	p.SelectRelated = selectRelated
	p.PrefetchRelated = prefetchRelated
}
