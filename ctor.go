package starbox

import (
	"fmt"
	"sync"
	"time"

	"bitbucket.org/ai69/amoy"
	"github.com/1set/starlet"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// StarlarkFunc is a function that can be called from Starlark.
type StarlarkFunc func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

// Starbox is a wrapper of starlet.Machine with additional features.
type Starbox struct {
	*starlet.Machine
	mu        sync.RWMutex
	hasRun    bool
	globals   starlet.StringAnyMap
	builtMods []string
	loadMods  map[string]starlet.ModuleLoader
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

// AddKeyValue adds a key-value pair to the global environment before running.
// If the key already exists, it will be overwritten.
// It panics if called after running.
func (s *Starbox) AddKeyValue(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add key-value pair after running")
	}
	if s.globals == nil {
		s.globals = make(starlet.StringAnyMap)
	}
	s.globals[key] = value
}

// AddKeyValues adds key-value pairs to the global environment before running.
// For each key-value pair, if the key already exists, it will be overwritten.
// It panics if called after running.
func (s *Starbox) AddKeyValues(keyValues starlet.StringAnyMap) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add key-value pairs after running")
	}
	if s.globals == nil {
		s.globals = make(starlet.StringAnyMap)
	}
	s.globals.Merge(keyValues)
}

// AddBuiltin adds a builtin function with name to the global environment before running.
// If the name already exists, it will be overwritten.
// It panics if called after running.
func (s *Starbox) AddBuiltin(name string, starFunc StarlarkFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add builtin after running")
	}
	sb := starlark.NewBuiltin(name, starFunc)
	s.Machine.AddGlobals(starlet.StringAnyMap{name: sb})
}

// AddNamedModules adds builtin modules by name to the preload and lazyload registry.
// It will not load the modules until the first run.
func (s *Starbox) AddNamedModules(moduleNames ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.builtMods = append(s.builtMods, moduleNames...)
}

// AddModuleLoader adds a custom module loader to the preload and lazyload registry.
// It will not load the module until the first run.
func (s *Starbox) AddModuleLoader(moduleName string, moduleLoader starlet.ModuleLoader) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.loadMods == nil {
		s.loadMods = make(map[string]starlet.ModuleLoader)
	}
	s.loadMods[moduleName] = moduleLoader
}

// AddModuleData creates a module for the given module data along with a module loader, and adds it to the preload and lazyload registry.
func (s *Starbox) AddModuleData(moduleName string, moduleData starlark.StringDict) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.loadMods == nil {
		s.loadMods = make(map[string]starlet.ModuleLoader)
	}
	s.loadMods[moduleName] = func() (starlark.StringDict, error) {
		sm := starlarkstruct.Module{Name: moduleName, Members: moduleData}
		return starlark.StringDict{
			moduleName: &sm,
		}, nil
	}
}

/*
pre, err := starlet.MakeBuiltinModuleLoaderList(moduleNames...)
if err != nil {
	panic(err)
}
lazy, err := starlet.MakeBuiltinModuleLoaderMap(moduleNames...)
if err != nil {
	panic(err)
}
s.Machine.SetPreloadModules(pre)
s.Machine.SetLazyloadModules(lazy)

SetPreloadModules ... by system names
SetLazyloadModules ... by system names ... exists?

SetCustomPreloadModules for real modules
SetCustomLazyloadModules for real modules
*/
