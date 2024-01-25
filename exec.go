package starbox

import (
	"io/fs"

	"github.com/1set/starlet"
	"github.com/psanford/memfs"
)

var (
	// defaultSafeModuleNames is the list of safe module names.
	defaultSafeModuleNames = []string{"base64", "go_idiomatic", "hashlib", "http", "json", "math", "random", "re", "struct", "time"}
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
	return s.Machine.Run()
}

// REPL starts a REPL session.
func (s *Starbox) REPL() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// prepare environment
	if err := s.prepareEnv(""); err != nil {
		return err
	}

	// run repl
	s.Machine.REPL()
	return nil
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
	// TODO: add preset builtin and distinct --- e.g. full, network, etc

	// preset module names
	if len(s.builtMods) > 0 {
		if preMods, err = starlet.MakeBuiltinModuleLoaderList(s.builtMods...); err != nil {
			return nil, nil, err
		}
		if lazyMods, err = starlet.MakeBuiltinModuleLoaderMap(s.builtMods...); err != nil {
			return nil, nil, err
		}
	}

	// custom module loaders
	if len(s.loadMods) > 0 {
		if preMods == nil {
			preMods = make(starlet.ModuleLoaderList, 0, len(s.loadMods))
		}
		if lazyMods == nil {
			lazyMods = make(starlet.ModuleLoaderMap, len(s.loadMods))
		}
		for name, loader := range s.loadMods {
			preMods = append(preMods, loader)
			lazyMods[name] = loader
		}
	}

	// result
	return preMods, lazyMods, nil
}
