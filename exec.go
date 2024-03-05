package starbox

import (
	"sort"
	"time"

	"github.com/1set/gut/yhash"
	"github.com/1set/gut/yrand"
	"github.com/1set/starlet"
	"github.com/psanford/memfs"
)

// Run executes a script and returns the converted output.
func (s *Starbox) Run(script string) (starlet.StringAnyMap, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// prepare environment
	if err := s.prepareEnv(script); err != nil {
		return nil, err
	}

	// run
	s.hasRun = true
	s.runTimes++
	return s.mac.Run()
}

// RunTimeout executes a script and returns the converted output.
func (s *Starbox) RunTimeout(script string, timeout time.Duration) (starlet.StringAnyMap, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// prepare environment
	if err := s.prepareEnv(script); err != nil {
		return nil, err
	}

	// run
	s.hasRun = true
	s.runTimes++
	return s.mac.RunWithTimeout(timeout, nil)
}

// REPL starts a REPL session.
func (s *Starbox) REPL() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// prepare environment -- no need to set script content
	if err := s.prepareEnv(""); err != nil {
		return err
	}

	// run
	s.hasRun = true
	s.runTimes++
	s.mac.REPL()
	return nil
}

// RunInspect executes a script and then REPL with result and returns the converted output.
func (s *Starbox) RunInspect(script string) (starlet.StringAnyMap, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// prepare environment
	if err := s.prepareEnv(script); err != nil {
		return nil, err
	}

	// run script
	s.hasRun = true
	s.runTimes++
	out, err := s.mac.Run()

	// repl
	s.mac.REPL()
	return out, err
}

// InspectCondFunc is a function type for inspecting the converted output of Run*() and decide whether to continue.
type InspectCondFunc func(starlet.StringAnyMap, error) bool

// RunInspectIf executes a script and then REPL with result and returns the converted output, if the condition is met.
// The condition function is called with the converted output and the error from Run*(), and returns true if REPL is needed.
func (s *Starbox) RunInspectIf(script string, cond InspectCondFunc) (starlet.StringAnyMap, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// prepare environment
	if err := s.prepareEnv(script); err != nil {
		return nil, err
	}

	// run script
	s.hasRun = true
	s.runTimes++
	out, err := s.mac.Run()

	// repl
	if cond(out, err) {
		s.mac.REPL()
	}
	return out, err
}

// Reset creates an new Starlet machine and keeps the settings.
func (s *Starbox) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	//s.mac.Reset()
	s.mac = newStarMachine(s.name)
	s.hasRun = false
}

func (s *Starbox) prepareEnv(script string) (err error) {
	// set script name and content
	setScriptContent := func(c string) {
		// generate script name from MD5 hash or random string as fallback
		var fn string
		if fn, _ = yhash.StringMD5(script); fn == "" {
			if fn, _ = yrand.StringBase36(12); fn == "" {
				fn = "box"
			}
		}
		s.mac.SetScript(fn+".star", []byte(c), s.modFS)
	}

	// if it's not the first run, set the script content only
	if s.hasRun {
		setScriptContent(script)
		return nil
	}

	// set custom tag and print function
	if s.structTag != "" {
		s.mac.SetCustomTag(s.structTag)
	}
	if s.printFunc != nil {
		s.mac.SetPrintFunc(s.printFunc)
	}

	// set variables
	s.mac.SetGlobals(s.globals)

	// extract module loaders
	preMods, lazyMods, err := s.extractModLoads()
	if err != nil {
		return err
	}

	// set modules to machine
	if len(preMods) > 0 || len(lazyMods) > 0 {
		s.mac.SetPreloadModules(preMods)
		s.mac.SetLazyloadModules(lazyMods)
	}

	// prepare script modules
	if len(s.scriptMods) > 0 && s.modFS == nil {
		rootFS := memfs.New()
		for fp, scr := range s.scriptMods {
			// TODO: support directory/file.star later
			if err := rootFS.WriteFile(fp, []byte(scr), 0644); err != nil {
				return err
			}
		}
		s.modFS = rootFS
	}

	// set script
	setScriptContent(script)

	// all is done
	return nil
}

func (s *Starbox) extractModLoads() (preMods starlet.ModuleLoaderList, lazyMods starlet.ModuleLoaderMap, err error) {
	// get modules by name: local module set + individual names for starlet
	var modNames []string
	if modNames, err = getModuleSet(s.modSet); err != nil {
		return nil, nil, err
	}
	modNames = append(modNames, s.builtMods...)
	modNames = uniqueStrings(modNames)

	// separate local module loaders from starlet module names
	var (
		letModNames []string
		modLoads    = make(starlet.ModuleLoaderMap, len(modNames))
	)
	for _, name := range modNames {
		if load, ok := localModuleLoaders[name]; ok {
			// for local module loaders
			modLoads[name] = load
		} else {
			// for starlet module names
			letModNames = append(letModNames, name)
		}
	}
	modNames = letModNames
	modLoads.Merge(s.loadMods) // custom module loaders overwrites local module loaders with the same name

	// convert starlet builtin module names to module loaders
	if len(modNames) > 0 {
		if preMods, err = starlet.MakeBuiltinModuleLoaderList(modNames...); err != nil {
			return nil, nil, err
		}
		if lazyMods, err = starlet.MakeBuiltinModuleLoaderMap(modNames...); err != nil {
			return nil, nil, err
		}
	}

	// merge custom module loaders
	if len(modLoads) > 0 {
		if preMods == nil {
			preMods = make(starlet.ModuleLoaderList, 0, len(modLoads))
		}
		if lazyMods == nil {
			lazyMods = make(starlet.ModuleLoaderMap, len(modLoads))
		}
		for name, loader := range modLoads {
			preMods = append(preMods, loader)
			lazyMods[name] = loader
		}
	}

	// result
	return preMods, lazyMods, nil
}

func uniqueStrings(ss []string) []string {
	if len(ss) < 2 {
		return ss
	}
	m := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		m[s] = struct{}{}
	}
	unique := make([]string, 0, len(m))
	for s := range m {
		unique = append(unique, s)
	}
	sort.Strings(unique)
	return unique
}
