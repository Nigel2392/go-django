package assert_test

import (
	"errors"
	"testing"

	"github.com/Nigel2392/django/core/assert"
)

func raises(t *testing.T) {
	if err := recover(); err == nil {
		t.Error("Expected panic, got nil")
	}
}

func notRaises(t *testing.T) {
	if err := recover(); err != nil {
		t.Errorf("Expected no panic, got %v", err)
	}
}

func panicIfErr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
}

func noPanic(f func(t *testing.T)) func(t *testing.T) {
	return func(t *testing.T) {
		assert.Panic = func(err error) {}
		f(t)
		assert.Panic = func(err error) {}
	}
}

func TestIsAssertionError(t *testing.T) {
	t.Run("PANIC", func(t *testing.T) {
		defer func() {
			var err any
			if err = recover(); err == nil {
				t.Error("Expected panic, got nil")
			}

			if !errors.Is(err.(error), assert.AssertionFailedError) {
				t.Errorf("Expected error to be an assertion error, got %v", err)
			}
		}()

		assert.Assert(false, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		var err = assert.Assert(false, "test")
		if !errors.Is(err, assert.AssertionFailedError) {
			t.Errorf("Expected error to be an assertion error, got %v", err)
		}
	}))

	t.Run("OK", func(t *testing.T) {
		var err = assert.Assert(true, "test")
		if err != nil {
			t.Errorf("Expected nil, got %v", err)
		}
	})
}

func TestFail(t *testing.T) {
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.Fail("test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.Fail("test")
	}))
}

func TestAssert(t *testing.T) {
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.Assert(false, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		if err := assert.Assert(false, "test"); err == nil {
			t.Error("Expected error, got nil")
		}
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.Assert(true, "test"))
	})
}

func TestEqual(t *testing.T) {
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.Equal(1, 2, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.Equal(1, 2, "test")
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.Equal(1, 1, "test"))
	})
}

func TestTruthy(t *testing.T) {
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.Truthy(false, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.Truthy(false, "test")
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.Truthy(true, "test"))
	})
}

func TestFalsy(t *testing.T) {
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.Falsy(true, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.Falsy(true, "test")
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.Falsy(false, "test"))
	})
}

func TestErr(t *testing.T) {
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.Err(errors.New("test"))
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.Err(errors.New("test"))
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.Err(nil))
	})
}

func TestErrNil(t *testing.T) {
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.ErrNil(nil)
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.ErrNil(errors.New("test"))
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.ErrNil(errors.New("test")))
	})
}

func TestGt(t *testing.T) {
	var slice = []int{1, 2, 3}
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.Gt(slice, 3, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.Gt(slice, 3, "test")
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.Gt(slice, 2, "test"))
	})
}

func TestLt(t *testing.T) {
	var slice = []int{1, 2, 3}
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.Lt(slice, 1, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.Lt(slice, 1, "test")
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.Lt(slice, 4, "test"))
	})
}

func TestGte(t *testing.T) {
	var slice = []int{1, 2, 3}
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.Gte(slice, 4, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.Gte(slice, 4, "test")
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.Gte(slice, 2, "test"))
	})
}

func TestLte(t *testing.T) {
	var slice = []int{1, 2, 3}
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.Lte(slice, 0, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.Lte(slice, 0, "test")
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.Lte(slice, 4, "test"))
	})
}

func TestContains(t *testing.T) {
	var slice = []int{1, 2, 3}
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.Contains(4, slice, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.Contains(4, slice, "test")
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.Contains(2, slice, "test"))
	})
}

func TestContainsFunc(t *testing.T) {
	var slice = []int{1, 2, 3}
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.ContainsFunc(slice, func(x int) bool { return x == 4 }, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.ContainsFunc(slice, func(x int) bool { return x == 4 }, "test")
	}))

	t.Run("OK", func(t *testing.T) {
		notRaises(t)
		panicIfErr(t, assert.ContainsFunc(slice, func(x int) bool { return x == 2 }, "test"))
	})
}

func TestTrue(t *testing.T) {
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.True(false, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.True(false, "test")
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.True(true, "test"))
	})
}

func TestFalse(t *testing.T) {
	t.Run("PANIC", func(t *testing.T) {
		defer raises(t)
		assert.False(true, "test")
	})

	t.Run("ERROR", noPanic(func(t *testing.T) {
		defer notRaises(t)
		assert.False(true, "test")
	}))

	t.Run("OK", func(t *testing.T) {
		defer notRaises(t)
		panicIfErr(t, assert.False(false, "test"))
	})
}
