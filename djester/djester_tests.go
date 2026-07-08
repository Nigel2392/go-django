//go:build test
// +build test

package djester

import "testing"

type Test interface {
	Name() string
	Test(dj *Tester, t *testing.T)
	Bench(dj *Tester, b *testing.B)
}

type BasicTest struct {
	Label     string
	Function  func(dj *Tester, t *testing.T)
	Benchmark func(dj *Tester, b *testing.B)
}

func (bt *BasicTest) Name() string {
	return bt.Label
}

func (bt *BasicTest) Test(dj *Tester, t *testing.T) {
	t.Helper()
	if bt.Function != nil {
		bt.Function(dj, t)
	}
}

func (bt *BasicTest) Bench(dj *Tester, b *testing.B) {
	b.Helper()
	if bt.Benchmark != nil {
		bt.Benchmark(dj, b)
	}
}
