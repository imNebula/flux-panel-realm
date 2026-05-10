package realm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerApplyStartAndStopByPidfile(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "realm")
	script := "#!/usr/bin/env sh\nif [ \"$1\" = \"--version\" ]; then echo realm-test; exit 0; fi\nwhile true; do sleep 1; done\n"
	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	m := NewManager(ManagerConfig{
		BinaryPath:  bin,
		ConfigDir:   filepath.Join(dir, "config"),
		LogDir:      filepath.Join(dir, "log"),
		PidFile:     filepath.Join(dir, "realm.pid"),
		ProcessName: "flux-realm-test",
		Instance:    "test",
		Mode:        "single",
	})
	ep := EndpointConfig{Name: "tcp", Listen: "127.0.0.1:18080", Remote: "127.0.0.1:18081"}
	res := m.ApplyConfig(Config{Endpoints: []EndpointConfig{ep}}, []string{"tcp"})
	if !res.Success || res.Action != "start" {
		t.Fatalf("apply failed: %#v", res)
	}
	pid, err := m.readPID()
	if err != nil || !processAlive(pid) {
		t.Fatalf("pidfile/process missing: pid=%d err=%v", pid, err)
	}
	res = m.DeleteServices([]string{"tcp"})
	if !res.Success || res.Action != "stop" {
		t.Fatalf("stop failed: %#v", res)
	}
	if _, err := os.Stat(m.cfg.PidFile); !os.IsNotExist(err) {
		t.Fatalf("pidfile still exists after stop")
	}
}

func TestManagerRejectsInvalidConfigWithoutReplacingCurrentHash(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "realm")
	if err := os.WriteFile(bin, []byte("#!/usr/bin/env sh\nif [ \"$1\" = \"--version\" ]; then echo realm-test; exit 0; fi\nwhile true; do sleep 1; done\n"), 0755); err != nil {
		t.Fatal(err)
	}
	m := NewManager(ManagerConfig{
		BinaryPath:  bin,
		ConfigDir:   filepath.Join(dir, "config"),
		LogDir:      filepath.Join(dir, "log"),
		PidFile:     filepath.Join(dir, "realm.pid"),
		ProcessName: "flux-realm-test",
		Instance:    "test",
		Mode:        "single",
	})
	good := EndpointConfig{Name: "tcp", Listen: "127.0.0.1:18082", Remote: "127.0.0.1:18083"}
	if res := m.ApplyConfig(Config{Endpoints: []EndpointConfig{good}}, []string{"tcp"}); !res.Success {
		t.Fatalf("initial apply failed: %#v", res)
	}
	defer m.DeleteServices([]string{"tcp"})
	before := m.ConfigHash()
	bad := EndpointConfig{Name: "bad", Listen: "not-a-listen", Remote: "127.0.0.1:18083"}
	res := m.ApplyConfig(Config{Endpoints: []EndpointConfig{bad}}, []string{"bad"})
	if res.Success {
		t.Fatalf("invalid config unexpectedly succeeded: %#v", res)
	}
	if after := m.ConfigHash(); after != before {
		t.Fatalf("config hash changed after rejected apply: before=%s after=%s", before, after)
	}
}
