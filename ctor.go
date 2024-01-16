package starbox

import (
	"fmt"
	"sync"
	"time"

	"bitbucket.org/ai69/amoy"
	"github.com/1set/starlet"
	"go.starlark.net/starlark"
)

// Starbox is a wrapper of starlet.Machine with additional features.
type Starbox struct {
	*starlet.Machine
	mu     sync.RWMutex
	hasRun bool
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
	return &Starbox{Machine: m}
}

// SetTag sets the custom tag of Go struct fields for Starlark.
func (s *Starbox) SetTag(tag string) {
	s.Machine.SetCustomTag(tag)
}

// SetKeyValue adds a key-value pair to the global environment before running.
// It panics if called after running.
func (s *Starbox) SetKeyValue(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add key-value pair after running")
	}
	s.Machine.AddGlobals(starlet.StringAnyMap{key: value})
}

// SetKeyValues adds key-value pairs to the global environment before running.
// It panics if called after running.
func (s *Starbox) SetKeyValues(keyValues starlet.StringAnyMap) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add key-value pairs after running")
	}
	s.Machine.AddGlobals(keyValues)
}
