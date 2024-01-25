package starbox

import (
	"io/fs"
	"sort"
	"time"

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
	return s.Machine.Run()
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
	return s.Machine.RunWithTimeout(timeout, nil)
}

// REPL starts a REPL session.
func (s *Starbox) REPL() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// prepare environment
	if err := s.prepareEnv(""); err != nil {
		return err
	}

	// run
	s.hasRun = true
	s.Machine.REPL()
	return nil
}

// Reset creates an new Starlet machine and keeps the settings.
func (s *Starbox) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	//s.Machine.Reset()
	s.Machine = newStarMachine(s.name)
	s.hasRun = false
}

func (s *Starbox) prepareEnv(script string) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// set variables
	s.Machine.SetGlobals(s.globals)

	// extract module loaders
	preMods, lazyMods, err := s.extractModuleLoaders()
	if err != nil {
		return err
	}

	// set modules to machine
	if len(preMods) > 0 || len(lazyMods) > 0 {
		s.Machine.SetPreloadModules(preMods)
		s.Machine.SetLazyloadModules(lazyMods)
	}

	// prepare script modules
	var modFS fs.FS
	if len(s.scriptMods) > 0 {
		rootFS := memfs.New()
		for fp, scr := range s.scriptMods {
			// TODO: support directory/file.star later
			if err := rootFS.WriteFile(fp, []byte(scr), 0644); err != nil {
				return err
			}
		}
		modFS = rootFS
	}

	// set script
	s.Machine.SetScript("box.star", []byte(script), modFS)

	// all is done
	return nil
}

func (s *Starbox) extractModuleLoaders() (preMods starlet.ModuleLoaderList, lazyMods starlet.ModuleLoaderMap, err error) {
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
		modLoads    starlet.ModuleLoaderMap
	)
	for _, name := range modNames {
		if load, ok := localModuleLoaders[name]; ok {
			if modLoads == nil {
				modLoads = make(starlet.ModuleLoaderMap, len(modNames))
			}
			modLoads[name] = load
		} else {
			letModNames = append(letModNames, name)
		}
	}
	modNames = letModNames
	modLoads.Merge(s.loadMods)

	// convert starlet builtin module names to module loaders
	if len(modNames) > 0 {
		if preMods, err = starlet.MakeBuiltinModuleLoaderList(modNames...); err != nil {
			return nil, nil, err
		}
		if lazyMods, err = starlet.MakeBuiltinModuleLoaderMap(modNames...); err != nil {
			return nil, nil, err
		}
	}

	// custom module loaders
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
