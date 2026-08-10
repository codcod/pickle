package flow

import "sort"

// registry is every flow definition this binary ships. "brine" is the only
// entry today; internal/config.Validate rejects any pickle.toml flow value
// that is not a key here, so ForName's fallback to Default below is never
// exercised on a config that has passed validation — see
// TestFlowNamesMatchConfigLegalValues, which is what keeps this registry and
// config's accepted set in agreement despite flow deliberately not importing
// config to check that itself.
var registry = map[string]*Definition{
	"brine": brine,
}

// Get returns the registered definition named name, if any.
func Get(name string) (*Definition, bool) {
	d, ok := registry[name]
	return d, ok
}

// Default is the flow used when no flow is configured — brine.
func Default() *Definition { return brine }

// Names lists every registered flow name, sorted.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ForName returns the definition registered under name, or Default() when
// name is not registered. A caller with a *config.Config should have already
// validated it (config.Validate rejects an unregistered flow name outright),
// so this fallback is a defensive default, not a routing decision made here.
func ForName(name string) *Definition {
	if d, ok := registry[name]; ok {
		return d
	}
	return Default()
}
