package mixins_test

import (
	"fmt"
	"testing"

	"github.com/Nigel2392/go-django/src/utils/mixins"
)

type DepthObj interface {
	GetDepth() int
}

type TestMixinObj struct {
	Depth int
	Objs  []DepthObj
}

func (t *TestMixinObj) GetDepth() int {
	return t.Depth
}

func (t *TestMixinObj) Mixins() []DepthObj {
	return t.Objs
}

type TestMixin TestMixinObj

func (t *TestMixin) GetDepth() int {
	return t.Depth
}

func (t *TestMixin) Mixins() []DepthObj {
	return t.Objs
}

type RootObj TestMixinObj

func (r *RootObj) GetDepth() int {
	return r.Depth
}

func (r *RootObj) Mixins() []DepthObj {
	return r.Objs
}

type mixinTest struct {
	tree         DepthObj
	expected     []int
	topDown      bool
	useTreeDepth bool
}

var mixinTests = []mixinTest{
	{
		tree: &RootObj{
			Depth: 0,
			Objs: []DepthObj{
				&TestMixinObj{
					Depth: 1,
					Objs: []DepthObj{
						&TestMixin{
							Depth: 2,
							Objs: []DepthObj{
								&TestMixinObj{
									Depth: 3,
									Objs:  []DepthObj{},
								},
								&TestMixinObj{
									Depth: 3,
									Objs:  []DepthObj{},
								},
								&TestMixinObj{
									Depth: 3,
									Objs:  []DepthObj{},
								},
							},
						},
						&TestMixin{
							Depth: 2,
							Objs: []DepthObj{
								&TestMixin{
									Depth: 3,
									Objs:  []DepthObj{},
								},
								&TestMixin{
									Depth: 3,
									Objs:  []DepthObj{},
								},
							},
						},
					},
				},
			},
		},
		expected: []int{
			0,
			1,
			2, 3, 3, 3,
			2, 3, 3,
		},
		topDown: true,
	},
	{
		tree: &RootObj{
			Depth: 0,
			Objs: []DepthObj{
				&TestMixinObj{
					Depth: 1,
					Objs: []DepthObj{
						&TestMixin{
							Depth: 2,
							Objs: []DepthObj{
								&TestMixinObj{
									Depth: 3,
									Objs:  []DepthObj{},
								},
								&TestMixinObj{
									Depth: 3,
									Objs:  []DepthObj{},
								},
								&TestMixinObj{
									Depth: 3,
									Objs:  []DepthObj{},
								},
							},
						},
						&TestMixin{
							Depth: 2,
							Objs: []DepthObj{
								&TestMixin{
									Depth: 3,
									Objs:  []DepthObj{},
								},
								&TestMixin{
									Depth: 3,
									Objs:  []DepthObj{},
								},
							},
						},
					},
				},
				&TestMixinObj{
					Depth: 1,
					Objs: []DepthObj{
						&TestMixin{
							Depth: 2,
							Objs: []DepthObj{
								&TestMixinObj{
									Depth: 3,
									Objs: []DepthObj{
										&TestMixinObj{
											Depth: 4,
											Objs: []DepthObj{
												&TestMixin{
													Depth: 5,
													Objs: []DepthObj{
														&TestMixinObj{
															Depth: 6,
															Objs:  []DepthObj{},
														},
														&TestMixinObj{
															Depth: 6,
															Objs:  []DepthObj{},
														},
														&TestMixinObj{
															Depth: 6,
															Objs:  []DepthObj{},
														},
													},
												},
												&TestMixin{
													Depth: 5,
													Objs: []DepthObj{
														&TestMixin{
															Depth: 6,
															Objs:  []DepthObj{},
														},
														&TestMixin{
															Depth: 6,
															Objs:  []DepthObj{},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		expected: []int{
			3, 3, 3, 2,
			3, 3, 2,
			1,
			6, 6, 6, 5,
			6, 6, 5,
			4,
			3,
			2,
			1,
			0,
		},
		topDown: false,
	},
	{
		tree: &RootObj{
			Depth: 0,
			Objs: []DepthObj{
				&TestMixinObj{
					Depth: 1,
					Objs: []DepthObj{
						&TestMixin{
							Depth: 2,
							Objs: []DepthObj{
								&TestMixinObj{
									Depth: 3,
									Objs:  []DepthObj{},
								},
								&TestMixinObj{
									Depth: 3,
									Objs:  []DepthObj{},
								},
								&TestMixinObj{
									Depth: 3,
									Objs:  []DepthObj{},
								},
							},
						},
						&TestMixin{
							Depth: 2,
							Objs: []DepthObj{
								&TestMixin{
									Depth: 3,
									Objs:  []DepthObj{},
								},
								&TestMixin{
									Depth: 3,
									Objs:  []DepthObj{},
								},
							},
						},
					},
				},
			},
		},
		expected: []int{
			0,
			1,
			2, 3, 3, 3,
			2, 3, 3,
		},
		topDown:      true,
		useTreeDepth: true,
	},
	{
		tree: &RootObj{
			Depth: 0,
			Objs: []DepthObj{
				&TestMixinObj{
					Depth: 1,
					Objs: []DepthObj{
						&TestMixin{
							Depth: 2,
							Objs: []DepthObj{
								&TestMixinObj{
									Depth: 3,
									Objs:  []DepthObj{},
								},
								&TestMixinObj{
									Depth: 3,
									Objs:  []DepthObj{},
								},
								&TestMixinObj{
									Depth: 3,
									Objs:  []DepthObj{},
								},
							},
						},
						&TestMixin{
							Depth: 2,
							Objs: []DepthObj{
								&TestMixin{
									Depth: 3,
									Objs:  []DepthObj{},
								},
								&TestMixin{
									Depth: 3,
									Objs:  []DepthObj{},
								},
							},
						},
					},
				},
				&TestMixinObj{
					Depth: 1,
					Objs: []DepthObj{
						&TestMixin{
							Depth: 2,
							Objs: []DepthObj{
								&TestMixinObj{
									Depth: 3,
									Objs: []DepthObj{
										&TestMixinObj{
											Depth: 4,
											Objs: []DepthObj{
												&TestMixin{
													Depth: 5,
													Objs: []DepthObj{
														&TestMixinObj{
															Depth: 6,
															Objs:  []DepthObj{},
														},
														&TestMixinObj{
															Depth: 6,
															Objs:  []DepthObj{},
														},
														&TestMixinObj{
															Depth: 6,
															Objs:  []DepthObj{},
														},
													},
												},
												&TestMixin{
													Depth: 5,
													Objs: []DepthObj{
														&TestMixin{
															Depth: 6,
															Objs:  []DepthObj{},
														},
														&TestMixin{
															Depth: 6,
															Objs:  []DepthObj{},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		expected: []int{
			3, 3, 3, 2,
			3, 3, 2,
			1,
			6, 6, 6, 5,
			6, 6, 5,
			4,
			3,
			2,
			1,
			0,
		},
		useTreeDepth: true,
		topDown:      false,
	},
}

func TestMixinTree(t *testing.T) {
	for i, test := range mixinTests {
		t.Run(fmt.Sprintf("BuildTree-%d", i), func(t *testing.T) {
			var tree = mixins.BuildMixinTree(test.tree)
			var got []int
			var err = tree.ForEach(test.topDown, func(t *mixins.MixinTree[DepthObj], depth int) error {
				if test.useTreeDepth {
					depth = t.Root.GetDepth()
				}
				got = append(got, depth)
				return nil
			})
			if err != nil {
				t.Errorf("Test %d: unexpected error: %v", i, err)
				return
			}

			if len(got) != len(test.expected) {
				t.Errorf("Test %d: expected length %d, got %d: %v != %v", i, len(test.expected), len(got), test.expected, got)
				return
			}

			for j := range got {
				if got[j] != test.expected[j] {
					t.Errorf("Test %d: at index %d, expected %d, got %d", i, j, test.expected[j], got[j])
				}
			}
		})
	}

	for i, test := range mixinTests {
		t.Run(fmt.Sprintf("BuildTree-%d", i), func(t *testing.T) {
			var got []int
			for mixin, depth := range mixins.Mixins(test.tree, test.topDown) {
				_ = mixin
				if test.useTreeDepth {
					depth = mixin.GetDepth()
				}
				got = append(got, depth)
			}

			if len(got) != len(test.expected) {
				t.Errorf("Test %d: expected length %d, got %d: %v != %v", i, len(test.expected), len(got), test.expected, got)
				return
			}

			for j := range got {
				if got[j] != test.expected[j] {
					t.Errorf("Test %d: at index %d, expected %d, got %d", i, j, test.expected[j], got[j])
				}
			}
		})
	}
}
