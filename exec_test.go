package starbox_test

import (
	"testing"

	"github.com/1set/starlet"
	"github.com/PureMature/starbox"
	"go.starlark.net/starlark"
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
	}

	for _, tt := range tests {
		t.Run("normal_"+tt.name, func(t *testing.T) {
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

func TestSetAddRunError(t *testing.T) {
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
		{
			name: "add invalid key value",
			fn: func(b *starbox.Starbox) {
				b.AddKeyValue("abc", make(chan int))
				//b.AddKeyValue("def cdf", 123)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := starbox.New("test")
			tt.fn(b)
			if out, err := b.Run(`z = 123`); err == nil {
				t.Errorf("expected error but not, output: %v", out)
			}
		})
	}
}
