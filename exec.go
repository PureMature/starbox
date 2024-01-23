package starbox

import "github.com/1set/starlet"

var (
	// defaultSafeModules is the list of safe modules.
	defaultSafeModules = []string{"base64", "go_idiomatic", "hashlib", "http", "json", "math", "random", "re", "struct", "time"}
)

func (s *Starbox) PrepareEnvironment() (err error) {
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

	// load it
	if len(preMods) > 0 {
		s.Machine.SetPreloadModules(preMods)
	}
	if len(lazyMods) > 0 {
		s.Machine.SetLazyloadModules(lazyMods)
	}

	// all is done
	return nil
}
