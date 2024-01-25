package starbox

import "fmt"

// ModuleSetName defines the name of a module set.
type ModuleSetName string

const (
	// EmptyModuleSet represents the predefined module set for empty scripts, it contains no modules.
	EmptyModuleSet ModuleSetName = "none"
	// SafeModuleSet represents the predefined module set for safe scripts, it contains only safe modules that do not have side effects with outside world.
	SafeModuleSet ModuleSetName = "safe"
	// NetworkModuleSet represents the predefined module set for network scripts, it's based on SafeModuleSet with additional network modules.
	NetworkModuleSet ModuleSetName = "network"
	// FullModuleSet represents the predefined module set for full scripts, it includes all available modules.
	FullModuleSet ModuleSetName = "full"
)

var (
	moduleSetMap = map[ModuleSetName][]string{
		EmptyModuleSet:   {},
		SafeModuleSet:    {"base64", "go_idiomatic", "hashlib", "json", "math", "random", "re", "struct", "time"},
		NetworkModuleSet: {"base64", "go_idiomatic", "hashlib", "http", "json", "math", "random", "re", "struct", "time"},
		FullModuleSet:    {"base64", "go_idiomatic", "hashlib", "http", "json", "math", "random", "re", "struct", "time"},
	}
)

func getModuleSet(modSet ModuleSetName) ([]string, error) {
	if mods, ok := moduleSetMap[modSet]; ok {
		return mods, nil
	}
	if modSet == "" {
		return []string{}, nil
	}
	return nil, fmt.Errorf("unknown module set: %s", modSet)
}
