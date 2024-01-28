package starbox

import (
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"time"

	"bitbucket.org/ai69/amoy"
	"github.com/1set/starlet"
	"github.com/1set/starlet/dataconv"
	"go.starlark.net/starlark"
)

// StarlarkFunc is a function that can be called from Starlark.
type StarlarkFunc func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

// Starbox is a wrapper of starlet.Machine with additional features.
type Starbox struct {
	mac        *starlet.Machine
	mu         sync.RWMutex
	hasRun     bool
	runTimes   uint
	name       string
	structTag  string
	printFunc  starlet.PrintFunc
	globals    starlet.StringAnyMap
	modSet     ModuleSetName
	builtMods  []string
	loadMods   starlet.ModuleLoaderMap
	scriptMods map[string]string
	modFS      fs.FS
}

// New creates a new Starbox instance with default settings.
func New(name string) *Starbox {
	return &Starbox{mac: newStarMachine(name), name: name}
}

func newStarMachine(name string) *starlet.Machine {
	m := starlet.NewDefault()
	m.EnableGlobalReassign()
	// m.SetInputConversionEnabled(false)
	// m.SetOutputConversionEnabled(true)
	m.SetPrintFunc(func(thread *starlark.Thread, msg string) {
		prefix := fmt.Sprintf("[⭐|%s](%s)", name, time.Now().UTC().Format(`15:04:05.000`))
		amoy.Eprintln(prefix, msg)
	})
	return m
}

// String returns the name of the Starbox instance.
func (s *Starbox) String() string {
	return fmt.Sprintf("🥡Box{name:%s,run:%d}", s.name, s.runTimes)
}

// GetMachine returns the underlying starlet.Machine instance.
func (s *Starbox) GetMachine() *starlet.Machine {
	return s.mac
}

// SetStructTag sets the custom tag of Go struct fields for Starlark.
// It panics if called after running.
func (s *Starbox) SetStructTag(tag string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot set tag after running")
	}
	s.structTag = tag
}

// SetPrintFunc sets the print function for Starlark.
// It panics if called after running.
func (s *Starbox) SetPrintFunc(printFunc starlet.PrintFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot set print function after running")
	}
	s.printFunc = printFunc
}

// SetModuleSet sets the module set to be loaded before running.
// It panics if called after running.
func (s *Starbox) SetModuleSet(modSet ModuleSetName) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot set module set after running")
	}
	s.modSet = modSet
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

// AddKeyValues adds key-value pairs to the global environment before running. Usually for output of Run()*.
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

// AddStarlarkValues adds key-value pairs to the global environment before running, the values are already converted to Starlark values.
// For each key-value pair, if the key already exists, it will be overwritten.
// It panics if called after running.
func (s *Starbox) AddStarlarkValues(keyValues starlark.StringDict) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add key-value pairs after running")
	}
	if s.globals == nil {
		s.globals = make(starlet.StringAnyMap)
	}
	for key, value := range keyValues {
		s.globals[key] = value
	}
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
	if s.globals == nil {
		s.globals = make(starlet.StringAnyMap)
	}
	sb := starlark.NewBuiltin(name, starFunc)
	s.globals[name] = sb
}

// AddNamedModules adds builtin modules by name to the preload and lazyload registry.
// It will not load the modules until the first run.
// It panics if called after running.
func (s *Starbox) AddNamedModules(moduleNames ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add named modules after running")
	}
	s.builtMods = append(s.builtMods, moduleNames...)
}

// AddModuleLoader adds a custom module loader to the preload and lazyload registry.
// It will not load the module until the first run, and load result can be accessed in script via load("module_name", "key1") or key1 directly.
// It panics if called after running.
func (s *Starbox) AddModuleLoader(moduleName string, moduleLoader starlet.ModuleLoader) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add module loader after running")
	}
	if s.loadMods == nil {
		s.loadMods = make(map[string]starlet.ModuleLoader)
	}
	s.loadMods[moduleName] = moduleLoader
}

// AddModuleData creates a module for the given module data along with a module loader, and adds it to the preload and lazyload registry.
// The given module data can be accessed in script via load("module_name", "key1") or module_name.key1.
// It panics if called after running.
func (s *Starbox) AddModuleData(moduleName string, moduleData starlark.StringDict) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add module data after running")
	}
	if s.loadMods == nil {
		s.loadMods = make(map[string]starlet.ModuleLoader)
	}
	s.loadMods[moduleName] = dataconv.WrapModuleData(moduleName, moduleData)
}

// AddModuleScript creates a module with given module script in virtual filesystem, and adds it to the preload and lazyload registry.
// The given module script can be accessed in script via load("module_name", "key1") or load("module_name.star", "key1") if module name has no ".star" suffix.
// It panics if called after running.
func (s *Starbox) AddModuleScript(moduleName, moduleScript string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add module script after running")
	}
	if s.scriptMods == nil {
		s.scriptMods = make(map[string]string)
	}
	name := strings.TrimSpace(moduleName)
	if !strings.HasSuffix(name, ".star") {
		name += ".star"
	}
	s.scriptMods[name] = moduleScript
}
