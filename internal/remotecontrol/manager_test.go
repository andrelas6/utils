package remotecontrol

import "testing"

// New fills in everything a Manager needs: the unit folder and home it's given,
// the real runner, and a way to find claude, so nothing is left nil.
func TestNew(t *testing.T) {
	m := New("/home/a/.config/systemd/user", "/home/a")

	if m.unitDir != "/home/a/.config/systemd/user" || m.home != "/home/a" {
		t.Errorf("folders = %q, %q", m.unitDir, m.home)
	}
	if _, ok := m.machine.(realMachine); !ok {
		t.Errorf("machine = %T, want realMachine", m.machine)
	}
	if m.lookPath == nil {
		t.Error("lookPath is nil")
	}
}
