package starbox

import (
	"io/fs"

	"github.com/1set/starlet"
	"github.com/psanford/memfs"
)

var (
	// defaultSafeModules is the list of safe modules.
	defaultSafeModules = []string{"base64", "go_idiomatic", "hashlib", "http", "json", "math", "random", "re", "struct", "time"}
)

func (s *Starbox) PrepareEnvironment(script string) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// variables
	// if s.globals == nil {
	// 	s.globals = make(starlet.StringAnyMap)
	// }
	s.Machine.SetGlobals(s.globals)

	// TODO: add prest builtins  --- e.g. full, network, etc

	// preset modules
	var (
		preMods  starlet.ModuleLoaderList
		lazyMods starlet.ModuleLoaderMap
	)
	if len(s.builtMods) > 0 {
		if preMods, err = starlet.MakeBuiltinModuleLoaderList(s.builtMods...); err != nil {
			return err
		}
		if lazyMods, err = starlet.MakeBuiltinModuleLoaderMap(s.builtMods...); err != nil {
			return err
		}
	}

	// custom modules
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

	// set modules to machine
	if len(preMods) > 0 || len(lazyMods) > 0 {
		s.Machine.SetPreloadModules(preMods)
		s.Machine.SetLazyloadModules(lazyMods)
	}

	// prepare script modules
	var modFS fs.FS
	if len(s.scriptMods) > 0 {
		rootFS := memfs.New()
		for name, script := range s.scriptMods {
			// TODO: support directory/file.star later
			if err := rootFS.WriteFile(name, []byte(script), 0644); err != nil {
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
