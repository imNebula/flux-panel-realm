package platform

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type Environment struct {
	OS                 string `json:"os"`
	Distro             string `json:"distro"`
	Version            string `json:"version"`
	Arch               string `json:"arch"`
	Libc               string `json:"libc"`
	InitSystem         string `json:"init_system"`
	ContainerType      string `json:"container_type"`
	IsRoot             bool   `json:"is_root"`
	HasCapNetAdmin     bool   `json:"cap_net_admin"`
	HasCapNetRaw       bool   `json:"cap_net_raw"`
	HasCapBPF          bool   `json:"cap_bpf"`
	HasIptables        bool   `json:"iptables"`
	HasNftables        bool   `json:"nftables"`
	TrafficStatsMethod string `json:"traffic_stats_method"`
	TrafficStatsReason string `json:"traffic_stats_reason"`
}

func Detect() Environment {
	env := Environment{
		OS:            runtime.GOOS,
		Arch:          normalizeArch(runtime.GOARCH),
		Libc:          detectLibc(),
		InitSystem:    detectInit(),
		ContainerType: detectContainer(),
		IsRoot:        os.Geteuid() == 0,
		HasIptables:   exists("iptables"),
		HasNftables:   exists("nft"),
	}
	env.Distro, env.Version = osRelease()
	env.HasCapNetAdmin, env.HasCapNetRaw, env.HasCapBPF = detectCaps()
	env.TrafficStatsMethod, env.TrafficStatsReason = selectTrafficMethod(env)
	return env
}

func normalizeArch(a string) string {
	switch a {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	case "386":
		return "i386"
	case "arm":
		return "armv7"
	default:
		return a
	}
}

func osRelease() (string, string) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return runtime.GOOS, ""
	}
	defer f.Close()
	vals := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		vals[k] = strings.Trim(v, `"`)
	}
	return first(vals["ID"], vals["NAME"], runtime.GOOS), first(vals["VERSION_ID"], vals["VERSION"])
}

func detectLibc() string {
	out, err := exec.Command("sh", "-c", "ldd --version 2>&1 || true").Output()
	if err == nil {
		l := strings.ToLower(string(out))
		if strings.Contains(l, "musl") {
			return "musl"
		}
		if strings.Contains(l, "glibc") || strings.Contains(l, "gnu libc") || strings.Contains(l, "debian") {
			return "glibc"
		}
	}
	if _, err := os.Stat("/lib/ld-musl-x86_64.so.1"); err == nil {
		return "musl"
	}
	return "unknown"
}

func detectInit() string {
	if exists("systemctl") {
		if err := exec.Command("systemctl", "is-system-running").Run(); err == nil {
			return "systemd"
		}
	}
	if _, err := os.Stat("/run/openrc"); err == nil || exists("rc-service") {
		return "openrc"
	}
	if exists("runsvdir") {
		return "runit"
	}
	if exists("s6-svscan") {
		return "s6"
	}
	if exists("supervisord") {
		return "supervisord"
	}
	return "none"
}

func detectContainer() string {
	if b, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		s := strings.ToLower(string(b))
		switch {
		case strings.Contains(s, "docker"):
			return "docker"
		case strings.Contains(s, "podman") || strings.Contains(s, "libpod"):
			return "podman"
		case strings.Contains(s, "lxc"):
			return "lxc"
		}
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "docker"
	}
	if b, err := os.ReadFile("/proc/vz/veinfo"); err == nil && len(b) >= 0 {
		return "openvz"
	}
	return "none"
}

func detectCaps() (bool, bool, bool) {
	b, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return false, false, false
	}
	var capEff string
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "CapEff:") {
			capEff = strings.TrimSpace(strings.TrimPrefix(line, "CapEff:"))
			break
		}
	}
	if capEff == "" {
		return false, false, false
	}
	if _, err := strconv.ParseUint(capEff, 16, 64); err != nil {
		return false, false, false
	}
	// Use shell arithmetic to avoid pulling a big capability dependency.
	has := func(bit int) bool {
		cmd := exec.Command("sh", "-c", "[ $((0x"+capEff+" & (1 << "+strconv.Itoa(bit)+"))) -ne 0 ]")
		return cmd.Run() == nil
	}
	return has(12), has(13), has(39)
}

func selectTrafficMethod(env Environment) (string, string) {
	if !env.IsRoot {
		return "none", "agent is not running as root"
	}
	if env.ContainerType == "lxc" || env.ContainerType == "openvz" {
		if !env.HasCapNetAdmin {
			return "none", env.ContainerType + " lacks CAP_NET_ADMIN"
		}
	}
	if env.HasNftables && env.HasCapNetAdmin {
		return "nftables", ""
	}
	if env.HasIptables && env.HasCapNetAdmin {
		return "iptables", ""
	}
	return "procfs", "firewall counters unavailable; procfs is node-level only"
}

func exists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
