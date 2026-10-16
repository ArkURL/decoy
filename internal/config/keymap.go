package config

// Action names used by the key map and config overrides.
const (
	Down        = "down"
	Up          = "up"
	PageDown    = "pagedown"
	PageUp      = "pageup"
	HalfDown    = "halfdown"
	HalfUp      = "halfup"
	NextChapter = "nextchapter"
	PrevChapter = "prevchapter"
	Top         = "top"
	Bottom      = "bottom"
	TOC         = "toc"
	Boss        = "boss"
	BossBack    = "bossback"
	BossFocus   = "bossfocus"
	Help        = "help"
	Quit        = "quit"
)

var defaultBindings = map[string][]string{
	Down:        {"j", "down"},
	Up:          {"k", "up"},
	PageDown:    {"n", "space", "pgdown", "f"},
	PageUp:      {"p", "pgup"},
	HalfDown:    {"ctrl+d", "d"},
	HalfUp:      {"ctrl+u", "u"},
	NextChapter: {"J", "]"},
	PrevChapter: {"K", "["},
	Top:         {"g", "home"},
	Bottom:      {"G", "end"},
	TOC:         {"t"},
	Boss:        {"b"},
	BossBack:    {"ctrl+b"},
	BossFocus:   {"i", "enter"},
	Help:        {"?"},
	Quit:        {"q", "ctrl+c"},
}

// KeyMap maps actions to the keys that trigger them.
type KeyMap struct {
	bindings map[string][]string
}

// NewKeyMap builds a key map from defaults plus per-action overrides.
func NewKeyMap(overrides map[string][]string) KeyMap {
	m := make(map[string][]string, len(defaultBindings))
	for k, v := range defaultBindings {
		m[k] = append([]string(nil), v...)
	}
	for action, keys := range overrides {
		if len(keys) == 0 {
			continue
		}
		m[action] = append([]string(nil), keys...)
	}
	return KeyMap{bindings: m}
}

// Has reports whether key triggers the given action.
func (k KeyMap) Has(action, key string) bool {
	for _, b := range k.bindings[action] {
		if b == key {
			return true
		}
	}
	return false
}

// Keys returns the keys bound to an action.
func (k KeyMap) Keys(action string) []string {
	return k.bindings[action]
}
