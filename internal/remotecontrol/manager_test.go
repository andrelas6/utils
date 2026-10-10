package remotecontrol

import "testing"

// New fills in everything a Manager needs: the unit folder and home it's given,
// the real runner, and a way to find claude, so nothing is left nil.
func TestNew(t *testing.T) {
	m := New("/home/a/.config/systemd/user", "/home/a")

	if m.UnitDir != "/home/a/.config/systemd/user" || m.Home != "/home/a" {
		t.Errorf("folders = %q, %q", m.UnitDir, m.Home)
	}
	if _, ok := m.Runner.(ExecRunner); !ok {
		t.Errorf("runner = %T, want ExecRunner", m.Runner)
	}
	if m.LookPath == nil {
		t.Error("LookPath is nil")
	}
}
