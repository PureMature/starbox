package starbox_test

import (
	"encoding/json"
	"testing"

	"github.com/1set/starlet"
	"github.com/PureMature/starbox"
)

func TestProbe(t *testing.T) {
	x := starlet.GetAllBuiltinModuleNames()
	xj, _ := json.Marshal(x)
	t.Log(string(xj))
}

func TestNew(t *testing.T) {
	b := starbox.New("test")
	n := `🥡Box{name:test,run:0}`
	if b.String() != n {
		t.Errorf("expect %s, got %s", n, b.String())
	}
}

func TestCreateAndRun(t *testing.T) {
	b := starbox.New("test")
	out, err := b.Run(`print('hello world')`)
	if err != nil {
		t.Error(err)
	}
	if out == nil {
		t.Error("expect not nil, got nil")
	}
	if len(out) != 1 {
		t.Errorf("expect 1, got %d", len(out))
	}
	if out["print"] != "hello world\n" {
		t.Errorf("expect 'hello world\n', got %s", out["print"])
	}
}
