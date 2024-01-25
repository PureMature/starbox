package starbox_test

import (
	"encoding/json"
	"testing"

	"github.com/1set/starlet"
)

func TestProbe(t *testing.T) {
	x := starlet.GetAllBuiltinModuleNames()
	xj, _ := json.Marshal(x)
	t.Log(string(xj))
}
