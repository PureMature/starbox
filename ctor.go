package starbox

import (
	"fmt"
	"time"

	"bitbucket.org/ai69/amoy"
	"github.com/1set/starlet"
	"go.starlark.net/starlark"
)

// Starbox is a wrapper of starlet.Machine with additional features.
type Starbox struct {
	*starlet.Machine
}

// NewStarbox creates a new Starbox instance with default settings.
func NewStarbox(name string) *Starbox {
	m := starlet.NewDefault()
	m.EnableGlobalReassign()
	// m.SetInputConversionEnabled(false)
	// m.SetOutputConversionEnabled(true)
	m.SetPrintFunc(func(thread *starlark.Thread, msg string) {
		prefix := fmt.Sprintf("[⭐|%s](%s)", name, time.Now().UTC().Format(`15:04:05.000`))
		amoy.Eprintln(prefix, msg)
	})
	return &Starbox{m}
}

// SetTag sets the custom tag of Go struct fields for Starlark.
func (s *Starbox) SetTag(tag string) {
	s.Machine.SetCustomTag(tag)
}
