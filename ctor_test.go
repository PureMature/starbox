package starbox_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"bitbucket.org/ai69/amoy"
	"github.com/1set/starlet"
	"github.com/PureMature/starbox"
	"go.starlark.net/starlark"
)

var (
	HereDoc   = amoy.HereDocf
	NoopPrint = func(thread *starlark.Thread, msg string) {
		return
	}
)

// TestProbe is a playground for exploring the external packages.
func TestProbe(t *testing.T) {
	x := starlet.GetAllBuiltinModuleNames()
	xj, _ := json.Marshal(x)
	t.Log(string(xj))
}

// TestNew tests the following:
// 1. Create a new Starbox instance.
// 2. Check the Stringer output.
// 3. Check the underlying starlet.Machine instance.
func TestNew(t *testing.T) {
	b := starbox.New("test")
	n := `🥡Box{name:test,run:0}`
	if b.String() != n {
		t.Errorf("expect %s, got %s", n, b.String())
	}
	m := b.GetMachine()
	if m == nil {
		t.Error("expect not nil, got nil")
	}
}

// TestCreateAndRun tests the following:
// 1. Create a new Starbox instance.
// 2. Run a script.
// 3. Check the output.
func TestCreateAndRun(t *testing.T) {
	b := starbox.New("test")
	out, err := b.Run(HereDoc(`
		s = 'Aloha!'
		print(s)
	`))
	if err != nil {
		t.Error(err)
	}
	if out == nil {
		t.Error("expect not nil, got nil")
	}
	if len(out) != 1 {
		t.Errorf("expect 1, got %d", len(out))
	}
	if es := "Aloha!"; out["s"] != es {
		t.Errorf("expect %q, got %q", es, out["s"])
	}
}

// TestSetStructTag tests the following:
// 1. Create a new Starbox instance.
// 2. Set the struct tag.
// 3. Run a script that uses the custom struct tag.
// 4. Check the output.
func TestSetStructTag(t *testing.T) {
	type testStruct struct {
		Nick1 string `json:"nick"`
		Nick2 string `starlark:"nick"`
	}
	s := testStruct{
		Nick1: "Kai",
		Nick2: "Kalani",
	}
	tests := []struct {
		tag      string
		expected string
	}{
		{"json", "Kai"},
		{"starlark", "Kalani"},
		{"", "Kalani"},
	}
	for i, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			b := starbox.New(fmt.Sprintf("test_%d", i))
			if tt.tag != "" {
				b.SetStructTag(tt.tag)
			}
			b.AddKeyValue("data", s)
			out, err := b.Run(HereDoc(`
				s = data.nick
				print(data)
			`))
			if err != nil {
				t.Error(err)
				return
			}
			if out == nil {
				t.Error("expect not nil, got nil")
				return
			}
			if len(out) != 1 {
				t.Errorf("expect 1, got %d", len(out))
				return
			}
			if es := tt.expected; out["s"] != es {
				t.Errorf("expect %q, got %q", es, out["s"])
			}
		})
	}

}

// TestSetPrintFunc tests the following:
// 1. Create a new Starbox instance.
// 2. Set the print function to output to a buffer.
// 3. Run a script that uses the print function.
// 4. Check the output.
func TestSetPrintFunc(t *testing.T) {
	var sb strings.Builder
	b := starbox.New("test")
	b.SetPrintFunc(func(thread *starlark.Thread, msg string) {
		sb.WriteString(msg)
	})
	out, err := b.Run(HereDoc(`
		print('Aloha!')
		print('Mahalo!')
	`))
	if err != nil {
		t.Error(err)
	}
	if out == nil {
		t.Error("expect not nil, got nil")
	}
	if len(out) != 0 {
		t.Errorf("expect 0, got %d", len(out))
	}
	actual := sb.String()
	expected := "Aloha!Mahalo!"
	if actual != expected {
		t.Errorf("expect %q, got %q", expected, actual)
	}
}

// TestSetModuleSet tests the following:
// 1. Create a new Starbox instance.
// 2. Set the module set.
// 3. Run a script that uses the module set.
// 4. Check the output.
func TestSetModuleSet(t *testing.T) {
	tests := []struct {
		setName starbox.ModuleSetName
		wantErr bool
		hasMod  []string
		nonMod  []string
	}{
		{
			setName: starbox.ModuleSetName("unknown"),
			wantErr: true,
		},
		{
			nonMod: []string{"base64", "json"},
		},
		{
			setName: starbox.EmptyModuleSet,
			nonMod:  []string{"base64", "json", "go_idiomatic"},
		},
		{
			setName: starbox.SafeModuleSet,
			hasMod:  []string{"base64", "json", "sleep"},
			nonMod:  []string{"http", "runtime", "go_idiomatic"},
		},
		{
			setName: starbox.NetworkModuleSet,
			hasMod:  []string{"base64", "json", "sleep", "http"},
			nonMod:  []string{"runtime", "go_idiomatic"},
		},
		{
			setName: starbox.FullModuleSet,
			hasMod:  []string{"base64", "json", "sleep", "http", "runtime"},
			nonMod:  []string{"go_idiomatic"},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			getBox := func() *starbox.Starbox {
				name := fmt.Sprintf("test_%d", i)
				b := starbox.New(name)
				b.SetModuleSet(tt.setName)
				b.SetPrintFunc(NoopPrint)
				return b
			}

			if tt.wantErr {
				b := getBox()
				_, err := b.Run(HereDoc(`a = 1`))
				if err == nil {
					t.Error("expect error, got nil")
				}
				return
			}

			// check for existing modules
			for _, m := range tt.hasMod {
				b := getBox()
				_, err := b.Run(HereDoc(fmt.Sprintf(`print(type(%s))`, m)))
				if err != nil {
					t.Errorf("expect nil for existing module %q, got %v", m, err)
					return
				}
			}

			// check for non-existing modules
			for _, m := range tt.nonMod {
				b := getBox()
				_, err := b.Run(HereDoc(fmt.Sprintf(`print(type(%s))`, m)))
				if err == nil {
					t.Errorf("expect error for non-existing module %q, got nil", m)
					return
				}
			}
		})
	}
}

// TestAddKeyValue tests the following:
// 1. Create a new Starbox instance.
// 2. Add a key-value pair.
// 3. Run a script that uses the key-value pair.
// 4. Check the output to see if the key-value pair is present.
func TestAddKeyValue(t *testing.T) {
	b := starbox.New("test")
	b.AddKeyValue("a", 10)
	out, err := b.Run(HereDoc(`b = a + 1`))
	if err != nil {
		t.Error(err)
	}
	if out == nil {
		t.Error("expect not nil, got nil")
	}
	if len(out) != 1 {
		t.Errorf("expect 1, got %d", len(out))
	}
	if es := int64(11); out["b"] != es {
		t.Errorf("expect %d, got %d", es, out["b"])
	}
}

// TestAddKeyValues tests the following:
// 1. Create a new Starbox instance.
// 2. Add key-value pairs.
// 3. Run a script that uses the key-value pairs.
// 4. Check the output to see if the key-value pairs are present.
func TestAddKeyValues(t *testing.T) {
	b := starbox.New("test")
	b.AddKeyValues(starlet.StringAnyMap{
		"a": 10,
		"b": 20,
	})
	out, err := b.Run(HereDoc(`c = a + b`))
	if err != nil {
		t.Error(err)
	}
	if out == nil {
		t.Error("expect not nil, got nil")
	}
	if len(out) != 1 {
		t.Errorf("expect 1, got %d", len(out))
	}
	if es := int64(30); out["c"] != es {
		t.Errorf("expect %d, got %d", es, out["c"])
	}
}
