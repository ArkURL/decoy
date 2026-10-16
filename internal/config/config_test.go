package config

import "testing"

func TestKeyMapOverride(t *testing.T) {
	k := NewKeyMap(map[string][]string{Boss: {"x"}})
	if !k.Has(Boss, "x") {
		t.Error("override not applied")
	}
	if k.Has(Boss, "b") {
		t.Error("default binding should be replaced by the override")
	}
	if !k.Has(Down, "j") || !k.Has(Quit, "q") {
		t.Error("unrelated defaults should be preserved")
	}
}

func TestSpaceIsPageDown(t *testing.T) {
	k := NewKeyMap(nil)
	// The space key reports as "space", not " ".
	if !k.Has(PageDown, "space") {
		t.Error(`PageDown should be bound to "space"`)
	}
	if !k.Has(PageUp, "pgup") {
		t.Error(`PageUp should be bound to "pgup"`)
	}
}

func TestDefaults(t *testing.T) {
	c := Default()
	if c.IdleSeconds != 30 {
		t.Errorf("IdleSeconds = %d, want 30", c.IdleSeconds)
	}
	if c.FakeTitle == "" {
		t.Error("FakeTitle should have a default")
	}
}
