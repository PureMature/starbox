package starbox

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"bitbucket.org/ai69/amoy"
	"github.com/1set/starlet"
	"github.com/1set/starlet/dataconv"
	libhttp "github.com/1set/starlet/lib/http"
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
	m.SetScriptCacheEnabled(true)
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

// SetFS sets the virtual filesystem for module scripts.
// If it's not nil, it'll override all the scripts added by AddModuleScript().
// It panics if called after running.
func (s *Starbox) SetFS(hfs fs.FS) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot set filesystem after running")
	}
	s.modFS = hfs
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

// AddKeyStarlarkValue adds a key-value pair to the global environment before running, the value is a Starlark value.
// If the key already exists, it will be overwritten.
// It panics if called after running.
func (s *Starbox) AddKeyStarlarkValue(key string, value starlark.Value) {
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

// AddModuleFunctions adds a module with the given module functions along with a module loader, and adds it to the preload and lazyload registry.
// The given module function can be accessed in script via load("module_name", "func1") or module_name.func1.
// It works like AddModuleData() but allows only functions as values.
// It panics if called after running.
func (s *Starbox) AddModuleFunctions(name string, funcs map[string]StarlarkFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add module function after running")
	}
	if s.loadMods == nil {
		s.loadMods = make(map[string]starlet.ModuleLoader)
	}
	sfd := starlark.StringDict{}
	for fn, fv := range funcs {
		sfd[fn] = starlark.NewBuiltin(name+"."+fn, fv)
	}
	s.loadMods[name] = dataconv.WrapModuleData(name, sfd)
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

// AddStructFunctions adds a module with the given struct functions along with a module loader, and adds it to the preload and lazyload registry.
// The given struct function can be accessed in script via load("struct_name", "func1") or struct_name.func1.
// It works like AddStructData() but allows only functions as values.
// It panics if called after running.
func (s *Starbox) AddStructFunctions(name string, funcs map[string]StarlarkFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add struct function after running")
	}
	if s.loadMods == nil {
		s.loadMods = make(map[string]starlet.ModuleLoader)
	}
	sfd := starlark.StringDict{}
	for fn, fv := range funcs {
		sfd[fn] = starlark.NewBuiltin(name+"."+fn, fv)
	}
	s.loadMods[name] = dataconv.WrapStructData(name, sfd)
}

// AddStructData creates a module for the given struct data along with a module loader, and adds it to the preload and lazyload registry.
// The given struct data can be accessed in script via load("struct_name", "key1") or struct_name.key1.
// It panics if called after running.
func (s *Starbox) AddStructData(structName string, structData starlark.StringDict) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add struct data after running")
	}
	if s.loadMods == nil {
		s.loadMods = make(map[string]starlet.ModuleLoader)
	}
	s.loadMods[structName] = dataconv.WrapStructData(structName, structData)
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

// AddHTTPContext adds HTTP request and response data wrapper to the global environment before running.
// It takes an HTTP request and returns the response data wrapper for setting response headers and body.
// It panics if called after running.
func (s *Starbox) AddHTTPContext(req *http.Request) *libhttp.ServerResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add HTTP context after running")
	}
	if s.globals == nil {
		s.globals = make(starlet.StringAnyMap)
	}

	// add request and response to globals
	s.globals["request"] = libhttp.ConvertServerRequest(req)
	resp := libhttp.NewServerResponse()
	s.globals["response"] = resp.Struct()
	return resp
}

const (
	memoryTypeName = "collective_memory"
)

// NewMemory creates a new shared dictionary for la mémoire collective.
func NewMemory() *dataconv.SharedDict {
	return dataconv.NewNamedSharedDict(memoryTypeName)
}

// AttachMemory adds a shared dictionary to the global environment before running.
func (s *Starbox) AttachMemory(name string, memory *dataconv.SharedDict) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add memory after running")
	}
	if s.globals == nil {
		s.globals = make(starlet.StringAnyMap)
	}
	s.globals[name] = memory
}

// CreateMemory creates a new shared dictionary for la mémoire collective with the given name, and adds it to the global environment before running.
func (s *Starbox) CreateMemory(name string) *dataconv.SharedDict {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRun {
		log.DPanic("cannot add memory after running")
	}
	if s.globals == nil {
		s.globals = make(starlet.StringAnyMap)
	}
	memory := dataconv.NewNamedSharedDict(memoryTypeName)
	s.globals[name] = memory
	return memory
}
