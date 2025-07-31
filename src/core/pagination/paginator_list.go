package pagination

import (
	"context"
	"errors"
)

// Paginator holds pagination logic
type Paginator[S ~[]E, E any] struct {
	Context    context.Context
	GetObject  func(E) E
	GetObjects func(amount int, offset int) (S, error)
	GetCount   func() (int, error)
	Amount     int
	cnt        int
}

func (p *Paginator[S, E]) NumPages() (int, error) {
	count, err := p.GetCount()
	if err != nil {
		return 0, err
	}
	return (count + p.PerPage() - 1) / p.PerPage(), nil
}

func (p *Paginator[S, E]) Count() (int, error) {
	if p.cnt == 0 {
		count, err := p.GetCount()
		if err != nil {
			return 0, err
		}
		p.cnt = count
	}
	return p.cnt, nil
}

func (p *Paginator[S, E]) PerPage() int {
	return p.Amount
}

func (p *Paginator[S, E]) Page(n int) (PageObject[E], error) {
	var amount = p.PerPage()

	if amount == 0 {
		return nil, errors.New("amount of objects per page cannot be 0")
	}

	var offset = (n - 1) * amount
	results, err := p.GetObjects(amount, offset)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, ErrNoResults
	}

	if p.GetObject != nil {
		for i, r := range results {
			results[i] = p.GetObject(r)
		}
	}

	return NewPageObject(p.Context, p, n-1, results), nil
}
