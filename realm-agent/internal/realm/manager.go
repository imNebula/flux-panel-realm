package realm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type ManagerConfig struct {
	BinaryPath  string
	ConfigDir   string
	LogDir      string
	PidFile     string
	ProcessName string
	Instance    string
	Mode        string
}

type ApplyResult struct {
	ApplyID          string   `json:"apply_id"`
	ChangedResources []string `json:"changed_resources"`
	ConfigHashBefore string   `json:"config_hash_before"`
	ConfigHashAfter  string   `json:"config_hash_after"`
	ValidationResult string   `json:"validation_result"`
	Action           string   `json:"action"`
	DurationMS       int64    `json:"duration_ms"`
	Success          bool     `json:"success"`
	ErrorMessage     string   `json:"error_message,omitempty"`
}

type Manager struct {
	cfg       ManagerConfig
	services  map[string]EndpointConfig
	chains    map[string]LegacyChain
	lastApply ApplyResult
}

func NewManager(cfg ManagerConfig) *Manager {
	return &Manager{
		cfg:      cfg,
		services: map[string]EndpointConfig{},
		chains:   map[string]LegacyChain{},
	}
}

func (m *Manager) Version() string {
	out, err := exec.Command(m.cfg.BinaryPath, "--version").CombinedOutput()
	if err != nil {
		return "unavailable: " + err.Error()
	}
	return strings.TrimSpace(string(out))
}

func (m *Manager) ConfigHash() string {
	b, err := os.ReadFile(m.configPath())
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (m *Manager) EndpointCount() int {
	n := 0
	for _, ep := range m.services {
		if !ep.Disabled {
			n++
		}
	}
	return n
}

func (m *Manager) LastApply() ApplyResult {
	return m.lastApply
}

func (m *Manager) Processes() []string {
	pid, err := m.readPID()
	if err != nil {
		return nil
	}
	if processAlive(pid) {
		return []string{fmt.Sprintf("%s(pid=%d)", m.cfg.ProcessName, pid)}
	}
	return nil
}

func (m *Manager) AddOrUpdateServices(raw []LegacyService) ApplyResult {
	var changed []string
	for _, s := range raw {
		ep, _, err := LegacyServiceToEndpoint(s, m.chains)
		if err != nil {
			return m.failure("validation", err)
		}
		m.services[s.Name] = ep
		changed = append(changed, s.Name)
	}
	return m.apply(changed)
}

func (m *Manager) DeleteServices(names []string) ApplyResult {
	for _, name := range names {
		delete(m.services, name)
	}
	return m.apply(names)
}

func (m *Manager) PauseServices(names []string) ApplyResult {
	for _, name := range names {
		ep := m.services[name]
		ep.Disabled = true
		m.services[name] = ep
	}
	return m.apply(names)
}

func (m *Manager) ResumeServices(names []string) ApplyResult {
	for _, name := range names {
		ep := m.services[name]
		ep.Disabled = false
		m.services[name] = ep
	}
	return m.apply(names)
}

func (m *Manager) AddOrUpdateChain(ch LegacyChain) ApplyResult {
	m.chains[ch.Name] = ch
	return m.apply([]string{ch.Name})
}

func (m *Manager) DeleteChain(name string) ApplyResult {
	delete(m.chains, name)
	return m.apply([]string{name})
}

func (m *Manager) ApplyConfig(cfg Config, changed []string) ApplyResult {
	next := map[string]EndpointConfig{}
	for i, ep := range cfg.Endpoints {
		if ep.Name == "" {
			ep.Name = fmt.Sprintf("endpoint-%d", i+1)
		}
		next[ep.Name] = ep
	}
	if err := Validate(DefaultConfig(filepath.Join(m.cfg.LogDir, "realm-"+m.cfg.Instance+".log"), endpointsFromMap(next))); err != nil {
		return m.failure("validation", err)
	}
	m.services = next
	return m.apply(changed)
}

func (m *Manager) apply(changed []string) ApplyResult {
	start := time.Now()
	before := m.ConfigHash()
	id := strconv.FormatInt(time.Now().UnixNano(), 36)
	logPath := filepath.Join(m.cfg.LogDir, "realm-"+m.cfg.Instance+".log")
	cfg := DefaultConfig(logPath, m.activeEndpoints())
	res := ApplyResult{ApplyID: id, ChangedResources: changed, ConfigHashBefore: before}
	if err := Validate(cfg); err != nil {
		res.ValidationResult = err.Error()
		res.Action = "none"
		res.DurationMS = time.Since(start).Milliseconds()
		res.Success = false
		res.ErrorMessage = err.Error()
		m.lastApply = res
		return res
	}
	res.ValidationResult = "ok"

	previous, _ := os.ReadFile(m.configPath())
	if err := m.writeAtomic(cfg); err != nil {
		return m.finishFailure(res, start, "write", err)
	}
	after := m.ConfigHash()
	res.ConfigHashAfter = after
	if before == after {
		res.Action = "none"
		res.Success = true
		res.DurationMS = time.Since(start).Milliseconds()
		m.lastApply = res
		return res
	}
	if len(cfg.Endpoints) == 0 {
		err := m.stop()
		res.Action = "stop"
		res.Success = err == nil
		if err != nil {
			res.ErrorMessage = err.Error()
		}
		res.DurationMS = time.Since(start).Milliseconds()
		m.lastApply = res
		return res
	}
	action := "restart"
	if _, err := m.readPID(); err != nil {
		action = "start"
	}
	if err := m.restart(); err != nil {
		if len(previous) > 0 {
			_ = os.WriteFile(m.configPath(), previous, 0644)
			_ = m.restart()
		}
		return m.finishFailure(res, start, action, err)
	}
	res.Action = action
	res.Success = true
	res.DurationMS = time.Since(start).Milliseconds()
	m.lastApply = res
	return res
}

func (m *Manager) activeEndpoints() []EndpointConfig {
	return endpointsFromMap(m.services)
}

func endpointsFromMap(services map[string]EndpointConfig) []EndpointConfig {
	var names []string
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]EndpointConfig, 0, len(names))
	for _, name := range names {
		ep := services[name]
		if !ep.Disabled && ep.Unsupported == "" {
			out = append(out, ep)
		}
	}
	return out
}

func (m *Manager) writeAtomic(cfg Config) error {
	if err := os.MkdirAll(m.cfg.ConfigDir, 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.configPath() + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, m.configPath()); err != nil {
		return err
	}
	dir, err := os.Open(m.cfg.ConfigDir)
	if err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	var check Config
	data, err := os.ReadFile(m.configPath())
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &check); err != nil {
		return err
	}
	return Validate(check)
}

func (m *Manager) restart() error {
	_ = m.stop()
	var cmd *exec.Cmd
	if bash, err := exec.LookPath("bash"); err == nil {
		cmd = exec.Command(bash, "-c", fmt.Sprintf("exec -a %s %s -c %s", shellQuote(m.cfg.ProcessName), shellQuote(m.cfg.BinaryPath), shellQuote(m.configPath())))
	} else {
		cmd = exec.Command(m.cfg.BinaryPath, "-c", m.configPath())
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = appendLog(filepath.Join(m.cfg.LogDir, "realm-"+m.cfg.Instance+".stdout.log"))
	cmd.Stderr = appendLog(filepath.Join(m.cfg.LogDir, "realm-"+m.cfg.Instance+".stderr.log"))
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := os.WriteFile(m.cfg.PidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		return err
	}
	time.Sleep(600 * time.Millisecond)
	if !processAlive(cmd.Process.Pid) {
		return fmt.Errorf("realm process exited after start")
	}
	return nil
}

func (m *Manager) stop() error {
	pid, err := m.readPID()
	if err != nil {
		return nil
	}
	if !processAlive(pid) {
		_ = os.Remove(m.cfg.PidFile)
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	_ = proc.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			_ = os.Remove(m.cfg.PidFile)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = proc.Signal(syscall.SIGKILL)
	_ = os.Remove(m.cfg.PidFile)
	return nil
}

func (m *Manager) readPID() (int, error) {
	b, err := os.ReadFile(m.cfg.PidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(b)))
}

func (m *Manager) configPath() string {
	return filepath.Join(m.cfg.ConfigDir, "realm.json")
}

func (m *Manager) finishFailure(res ApplyResult, start time.Time, action string, err error) ApplyResult {
	res.Action = action
	res.DurationMS = time.Since(start).Milliseconds()
	res.Success = false
	res.ErrorMessage = err.Error()
	m.lastApply = res
	return res
}

func (m *Manager) failure(action string, err error) ApplyResult {
	res := ApplyResult{
		ApplyID:      strconv.FormatInt(time.Now().UnixNano(), 36),
		Action:       action,
		Success:      false,
		ErrorMessage: err.Error(),
	}
	m.lastApply = res
	return res
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

func appendLog(path string) io.Writer {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return os.Stderr
	}
	return f
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
