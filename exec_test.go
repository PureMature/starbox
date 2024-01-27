package starbox_test

import (
	"testing"
	"time"

	"github.com/1set/starlet"
	"github.com/PureMature/starbox"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.uber.org/zap"
)

func TestSimpleRun(t *testing.T) {
	b := starbox.New("test")
	out, err := b.Run(`s = "hello world"; print(s)`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["s"] != "hello world" {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestEmptyRun(t *testing.T) {
	b := starbox.New("test")
	out, err := b.Run(``)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestRunTimeout(t *testing.T) {
	// timeout
	b := starbox.New("test")
	b.SetModuleSet(starbox.SafeModuleSet)
	out, err := b.RunTimeout(`sleep(1.5)`, time.Second)
	if err == nil {
		t.Errorf("expected error but not, output: %v", out)
	}

	// no timeout
	b.Reset()
	out, err = b.RunTimeout(`sleep(0.2)`, time.Second)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunTwice(t *testing.T) {
	b := starbox.New("test")
	out, err := b.Run(`a = 10`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["a"] != int64(10) {
		t.Errorf("unexpected output: %v", out)
	}

	out, err = b.Run(`b = a << 2`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["b"] != int64(40) {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestRunTimeoutTwice(t *testing.T) {
	b := starbox.New("test")
	out, err := b.RunTimeout(`a = 10`, time.Second)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["a"] != int64(10) {
		t.Errorf("unexpected output: %v", out)
	}

	out, err = b.RunTimeout(`b = a << 2`, time.Second)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["b"] != int64(40) {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestRunWithPreviousResult(t *testing.T) {
	b1 := starbox.New("test1")
	out, err := b1.Run(HereDoc(`
		a = 10; b = 20; c = 30

		def mul(*args):
			v = 1
			for a in args:
				v *= a
			return v
	`))
	if err != nil {
		t.Errorf("unexpected error1: %v", err)
	}
	if out["a"] != int64(10) || out["b"] != int64(20) || out["c"] != int64(30) {
		t.Errorf("unexpected output1: %v", out)
	}

	b2 := starbox.New("test2")
	b2.AddKeyValues(out)
	out, err = b2.Run(`d = a + b + c + mul(a, b, c)`)
	if err != nil {
		t.Errorf("unexpected error2: %v", err)
	}
	if out["d"] != int64(6060) {
		t.Errorf("unexpected output2: %v", out)
	}
}

// TestREPL tests the following:
// 1. Create a new Starbox instance.
// 2. Run the REPL.
func TestREPL(t *testing.T) {
	b := starbox.New("test")
	if err := b.REPL(); err != nil {
		t.Error(err)
	}
}

// TestRunInspect tests the following:
// 1. Create a new Starbox instance.
// 2. Run a script that uses the inspect function.
// 3. Check the output.
func TestRunInspect(t *testing.T) {
	b1 := starbox.New("test1")
	out, err := b1.RunInspect(HereDoc(`
		a = 123
		s = invalid(1)
	`))
	if err == nil {
		t.Errorf("expected error but not, output: %v", out)
	}
	t.Logf("output1: %v", out)

	b2 := starbox.New("test2")
	out, err = b2.RunInspect(HereDoc(`
		a = 456
	`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	t.Logf("output2: %v", out)
}

func TestSetAddRunPanic(t *testing.T) {
	getBox := func(t *testing.T) *starbox.Starbox {
		b := starbox.New("test")
		out, err := b.Run(`s = "hello world"`)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if out["s"] != "hello world" {
			t.Errorf("unexpected output: %v", out)
		}
		logger, err := zap.NewDevelopment()
		if err != nil {
			t.Errorf("unexpected error for zap: %v", err)
		}
		starbox.SetLog(logger.Sugar())
		return b
	}

	tests := []struct {
		name string
		fn   func(b *starbox.Starbox)
	}{
		{
			name: "set struct",
			fn: func(b *starbox.Starbox) {
				b.SetStructTag("json")
			},
		},
		{
			name: "set printf",
			fn: func(b *starbox.Starbox) {
				b.SetPrintFunc(func(thread *starlark.Thread, msg string) {
					t.Logf("printf: %s", msg)
				})
			},
		},
		{
			name: "set module set",
			fn: func(b *starbox.Starbox) {
				b.SetModuleSet(starbox.SafeModuleSet)
			},
		},
		{
			name: "add key value",
			fn: func(b *starbox.Starbox) {
				b.AddKeyValue("a", 1)
			},
		},
		{
			name: "add key values",
			fn: func(b *starbox.Starbox) {
				b.AddKeyValues(starlet.StringAnyMap{
					"a": 1,
					"b": 2,
				})
			},
		},
		{
			name: "add builtin",
			fn: func(b *starbox.Starbox) {
				b.AddBuiltin("a", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
					return starlark.String("aloha"), nil
				})
			},
		},
		{
			name: "add named module",
			fn: func(b *starbox.Starbox) {
				b.AddNamedModules("base64")
			},
		},
		{
			name: "add module loader",
			fn: func(b *starbox.Starbox) {
				b.AddModuleLoader("mine", func() (starlark.StringDict, error) {
					return starlark.StringDict{
						"shift": starlark.NewBuiltin("shift", func(thread *starlark.Thread, bt *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
							var a, b int64
							if err := starlark.UnpackArgs(bt.Name(), args, kwargs, "a", &a, "b", &b); err != nil {
								return nil, err
							}
							return starlark.MakeInt64(a << b).Add(starlark.MakeInt(5)), nil
						}),
						"num": starlark.MakeInt(100),
					}, nil
				})
			},
		},
		{
			name: "add module data",
			fn: func(b *starbox.Starbox) {
				b.AddModuleData("data", starlark.StringDict{
					"a": starlark.MakeInt(10),
					"b": starlark.MakeInt(20),
					"c": starlark.MakeInt(300),
				})
			},
		},
		{
			name: "add module script",
			fn: func(b *starbox.Starbox) {
				b.AddModuleScript("data", HereDoc(`
					a = 10
					b = 20
					c = 300
				`))
			},
		},
		{
			name: "add module script using module",
			fn: func(b *starbox.Starbox) {
				b.AddNamedModules("base64")
				b.AddModuleScript("data", HereDoc(`
					load("base64", "encode")
					a = encode("hello world")
					print(a, base64.encode("Aloha!"))
				`))
			},
		},
	}

	for _, tt := range tests {
		t.Run("before_"+tt.name, func(t *testing.T) {
			b := starbox.New("test")
			_, err := b.Run(`z = 123`)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	for _, tt := range tests {
		t.Run("after_"+tt.name, func(t *testing.T) {
			box := getBox(t)
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic but not")
				}
			}()
			tt.fn(box)
		})
	}
}

func TestSetAddPrepareError(t *testing.T) {
	tests := []struct {
		name string
		fn   func(b *starbox.Starbox)
	}{
		{
			name: "set invalid module set",
			fn: func(b *starbox.Starbox) {
				b.SetModuleSet("missing")
			},
		},
		{
			name: "add empty named module",
			fn: func(b *starbox.Starbox) {
				b.AddNamedModules("")
			},
		},
		{
			name: "add invalid named module",
			fn: func(b *starbox.Starbox) {
				b.AddNamedModules("dont_exist")
			},
		},
		{
			name: "add invalid module script",
			fn: func(b *starbox.Starbox) {
				b.AddModuleScript("///", HereDoc(`
					a = 10
					b = 20
					c = 300
				`))
			},
		},
	}
	for _, tt := range tests {
		t.Run("normal_"+tt.name, func(t *testing.T) {
			b := starbox.New("test")
			tt.fn(b)
			if out, err := b.Run(`z = 123`); err == nil {
				t.Errorf("expected error but not, output: %v", out)
			}
		})
	}
	for _, tt := range tests {
		t.Run("timeout_"+tt.name, func(t *testing.T) {
			b := starbox.New("test")
			tt.fn(b)
			if out, err := b.RunTimeout(`z = 123`, time.Second); err == nil {
				t.Errorf("expected error but not, output: %v", out)
			}
		})
	}
	for _, tt := range tests {
		t.Run("repl_"+tt.name, func(t *testing.T) {
			b := starbox.New("test")
			tt.fn(b)
			if err := b.REPL(); err == nil {
				t.Errorf("expected error but not")
			}
		})
	}
	for _, tt := range tests {
		t.Run("inspect_"+tt.name, func(t *testing.T) {
			b := starbox.New("test")
			tt.fn(b)
			if out, err := b.RunInspect(`z = 123`); err == nil {
				t.Errorf("expected error but not, output: %v", out)
			}
		})
	}
}

func TestSetAddRunError(t *testing.T) {
	tests := []struct {
		name string
		fn   func(b *starbox.Starbox)
	}{
		{
			name: "add invalid key value",
			fn: func(b *starbox.Starbox) {
				b.AddKeyValue("abc", make(chan int))
				//b.AddKeyValue("def cdf", 123)
			},
		},
	}
	for _, tt := range tests {
		t.Run("normal_"+tt.name, func(t *testing.T) {
			b := starbox.New("test")
			tt.fn(b)
			if out, err := b.Run(`z = 123`); err == nil {
				t.Errorf("expected error but not, output: %v", out)
			}
		})
	}
	for _, tt := range tests {
		t.Run("timeout_"+tt.name, func(t *testing.T) {
			b := starbox.New("test")
			tt.fn(b)
			if out, err := b.RunTimeout(`z = 123`, time.Second); err == nil {
				t.Errorf("expected error but not, output: %v", out)
			}
		})
	}
	for _, tt := range tests {
		t.Run("inspect_"+tt.name, func(t *testing.T) {
			b := starbox.New("test")
			tt.fn(b)
			if out, err := b.RunInspect(`z = 123`); err == nil {
				t.Errorf("expected error but not, output: %v", out)
			}
		})
	}
}

func TestOverrideKeyValue(t *testing.T) {
	b := starbox.New("test")
	b.AddKeyValue("a", 1)
	b.AddKeyValue("a", 20)
	b.AddKeyValue("a", 300)
	out, err := b.Run(`res = a`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["res"] != int64(300) {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestOverrideKeyValues(t *testing.T) {
	b := starbox.New("test")
	b.AddKeyValues(starlet.StringAnyMap{
		"a": 1,
		"b": 2,
	})
	b.AddKeyValues(starlet.StringAnyMap{
		"a": 10,
		"b": 20,
	})
	b.AddKeyValues(starlet.StringAnyMap{
		"a": 100,
		"b": 200,
	})
	out, err := b.Run(`res = a + b`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["res"] != int64(300) {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestOverrideBuiltin(t *testing.T) {
	b := starbox.New("test")
	b.AddBuiltin("a", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return starlark.String("aloha"), nil
	})
	b.AddBuiltin("a", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return starlark.String("hello"), nil
	})
	out, err := b.Run(`res = a()`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["res"] != "hello" {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestOverrideModuleLoader(t *testing.T) {
	b := starbox.New("test")
	b.AddModuleLoader("mine", func() (starlark.StringDict, error) {
		return starlark.StringDict{
			"shift": starlark.NewBuiltin("shift", func(thread *starlark.Thread, bt *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
				var a, b int64
				if err := starlark.UnpackArgs(bt.Name(), args, kwargs, "a", &a, "b", &b); err != nil {
					return nil, err
				}
				return starlark.MakeInt64(a << b).Add(starlark.MakeInt(5)), nil
			}),
			"num": starlark.MakeInt(100),
		}, nil
	})
	b.AddModuleLoader("mine", func() (starlark.StringDict, error) {
		return starlark.StringDict{
			"shift": starlark.NewBuiltin("shift", func(thread *starlark.Thread, bt *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
				var a, b int64
				if err := starlark.UnpackArgs(bt.Name(), args, kwargs, "a", &a, "b", &b); err != nil {
					return nil, err
				}
				return starlark.MakeInt64(a << b).Add(starlark.MakeInt(10)), nil
			}),
			"num": starlark.MakeInt(200),
		}, nil
	})
	out, err := b.Run(`res = shift(10, 2)`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["res"] != int64(50) {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestOverrideModuleData(t *testing.T) {
	b := starbox.New("test")
	b.AddModuleData("data", starlark.StringDict{
		"a": starlark.MakeInt(10),
		"b": starlark.MakeInt(20),
		"c": starlark.MakeInt(300),
	})
	b.AddModuleData("data", starlark.StringDict{
		"a": starlark.MakeInt(100),
		"b": starlark.MakeInt(200),
		"c": starlark.MakeInt(3000),
	})
	out, err := b.Run(`res = data.a + data.b + data.c`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["res"] != int64(3300) {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestOverrideModuleScript(t *testing.T) {
	b := starbox.New("test")
	b.AddModuleScript("data", HereDoc(`
		a = 10
		b = 20
		c = 300
	`))
	b.AddModuleScript("data", HereDoc(`
		a = 100
		b = 200
		c = 3000
	`))
	out, err := b.Run(`load("data", "a", "b", "c"); res = a + b + c`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["res"] != int64(3300) {
		t.Errorf("unexpected output: %v", out)
	}
}

// TestConflictGlobalModule tests if the global variable is overridden by the module data.
// Since the extra variables in Starlet are not set, the module data will override the global variable, and there's no way to override the module data.
func TestConflictGlobalModule(t *testing.T) {
	b := starbox.New("test")
	b.AddNamedModules("go_idiomatic")
	b.AddKeyValues(starlet.StringAnyMap{
		"bin": 1024,
		"hex": "0x400",
	})
	// check if the module is loaded and the member is overridden
	out, err := b.Run(HereDoc(`
		print(type(bin), type(hex), type(sum))
		# res = sum([bin, 10]); x = hex + " " + str(bin)
		x = bin(10) + " " + hex(2048)
	`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if es := `0b1010 0x800`; out["x"] != es {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestConflictModuleMemberLoader(t *testing.T) {
	name := "go_idiomatic"
	b := starbox.New("test")
	b.AddNamedModules(name)
	b.AddModuleLoader(name, func() (starlark.StringDict, error) {
		return starlark.StringDict{
			"bin": starlark.MakeInt(1024),
		}, nil
	})
	// check if the module is loaded and the member is overridden
	out, err := b.Run(`res = sum([bin, 10])`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["res"] != int64(1034) {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestConflictModuleStructLoader(t *testing.T) {
	name := "base64"
	b := starbox.New("test")
	b.AddNamedModules(name)
	b.AddModuleLoader(name, func() (starlark.StringDict, error) {
		return starlark.StringDict{
			name: &starlarkstruct.Module{
				Name: name,
				Members: starlark.StringDict{
					"shift": starlark.NewBuiltin("shift", func(thread *starlark.Thread, bt *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
						var a, b int64
						if err := starlark.UnpackArgs(bt.Name(), args, kwargs, "a", &a, "b", &b); err != nil {
							return nil, err
						}
						return starlark.MakeInt64(a << b).Add(starlark.MakeInt(5)), nil
					}),
					"num": starlark.MakeInt(100),
				},
			},
			"num": starlark.MakeInt(1000),
		}, nil
	})
	// check if the module is loaded
	out, err := b.Run(`res = base64.shift(10, 2) + base64.num + num; print(dir(base64))`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["res"] != int64(1145) {
		t.Errorf("unexpected output: %v", out)
	}
}
