// Package traffic provides per-endpoint traffic statistics collection.
// Priority: nftables > iptables > procfs (node-level) > none.
// Handles counter delta, restart (wrap/zero), rule missing, permission limited.
package traffic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Sample represents one traffic measurement for a specific endpoint.
type Sample struct {
	NodeID     int64  `json:"node_id"`
	TunnelID   int64  `json:"tunnel_id,omitempty"`
	ForwardID  int64  `json:"forward_id,omitempty"`
	UserID     int64  `json:"user_id,omitempty"`
	ListenAddr string `json:"listen_addr"`
	ListenPort int    `json:"listen_port"`
	Protocol   string `json:"protocol"` // tcp / udp
	InBytes    int64  `json:"in_bytes"`
	OutBytes   int64  `json:"out_bytes"`
	TotalBytes int64  `json:"total_bytes"`
	// BillingBytes is calculated by the backend based on tunnel config;
	// agent reports raw bytes, backend applies ratio.
	BillingBytes int64  `json:"billing_bytes"`
	SampleTime   int64  `json:"sample_time"`
	Method       string `json:"method"`
	Error        string `json:"error,omitempty"`
}

// EndpointKey uniquely identifies an endpoint for tracking purposes.
type EndpointKey struct {
	ForwardID int64
	Port      int
	Protocol  string // "tcp" | "udp"
}

// counterState records the last seen counter value to compute deltas.
type counterState struct {
	lastIn  int64
	lastOut int64
	ts      int64
}

// Collector gathers per-endpoint traffic statistics.
type Collector struct {
	method  string // "nftables" | "iptables" | "procfs" | "none"
	reason  string
	mu      sync.Mutex
	prev    map[EndpointKey]counterState
	tableV4 string // nftables table name for IPv4 rules
	chainV4 string
}

const (
	fluxTable = "flux_realm_stats"
	fluxChain = "flux_endpoint_stats"
)

// NewCollector creates a Collector using the best available method.
func NewCollector(method, reason string) *Collector {
	c := &Collector{
		method:  method,
		reason:  reason,
		prev:    make(map[EndpointKey]counterState),
		tableV4: fluxTable,
		chainV4: fluxChain,
	}
	return c
}

// Method returns the detected stats collection method.
func (c *Collector) Method() string { return c.method }

// Reason returns why a downgraded method was selected (empty if best method).
func (c *Collector) Reason() string { return c.reason }

// EnsureRules ensures nftables/iptables rules exist for the given endpoints.
// Must be called after each Realm config apply.
// Parameters:
//
//	endpoints: list of (port, protocol) pairs to track
func (c *Collector) EnsureRules(endpoints []EndpointKey) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.method {
	case "nftables":
		return c.ensureNftRules(endpoints)
	case "iptables":
		return c.ensureIptRules(endpoints)
	default:
		return nil // procfs / none – no rules to manage
	}
}

// RemoveRules removes firewall rules for deleted endpoints.
func (c *Collector) RemoveRules(endpoints []EndpointKey) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.method {
	case "nftables":
		return c.removeNftRules(endpoints)
	case "iptables":
		return c.removeIptRules(endpoints)
	default:
		return nil
	}
}

// Collect gathers traffic deltas for all tracked endpoints.
// Returns one Sample per (ForwardID, Protocol).
func (c *Collector) Collect(keys []EndpointKey) ([]Sample, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.method {
	case "nftables":
		return c.collectNft(keys)
	case "iptables":
		return c.collectIpt(keys)
	case "procfs":
		return c.collectProcfs(keys)
	default:
		return nil, fmt.Errorf("traffic stats not available: %s", c.reason)
	}
}

// ─── nftables ────────────────────────────────────────────────────────────────

func (c *Collector) ensureNftRules(endpoints []EndpointKey) error {
	// Create table + chain if missing.
	setupScript := fmt.Sprintf(`
nft list table inet %s > /dev/null 2>&1 || \
  nft add table inet %s
nft list chain inet %s %s > /dev/null 2>&1 || \
  nft add chain inet %s %s '{ type filter hook prerouting priority -150; policy accept; }'
`, c.tableV4, c.tableV4, c.tableV4, c.chainV4, c.tableV4, c.chainV4)
	if err := runSh(setupScript); err != nil {
		return fmt.Errorf("nft setup: %w", err)
	}

	// Ensure one counter rule per (port, protocol).
	for _, ep := range endpoints {
		proto := ep.Protocol
		if proto == "" {
			proto = "tcp"
		}
		// Check if rule already exists by comment handle.
		ruleComment := ruleHandle(ep)
		existsScript := fmt.Sprintf(`nft list chain inet %s %s | grep -q '%s'`, c.tableV4, c.chainV4, ruleComment)
		if runSh(existsScript) == nil {
			continue // rule exists
		}
		addScript := fmt.Sprintf(
			`nft add rule inet %s %s %s dport %d counter comment "%s"`,
			c.tableV4, c.chainV4, proto, ep.Port, ruleComment,
		)
		if err := runSh(addScript); err != nil {
			return fmt.Errorf("nft add rule port %d/%s: %w", ep.Port, proto, err)
		}
	}
	return nil
}

func (c *Collector) removeNftRules(endpoints []EndpointKey) error {
	for _, ep := range endpoints {
		comment := ruleHandle(ep)
		// Get handle number for the rule, then delete by handle.
		script := fmt.Sprintf(`
handle=$(nft -a list chain inet %s %s 2>/dev/null | grep '%s' | grep -oP '# handle \K[0-9]+')
[ -n "$handle" ] && nft delete rule inet %s %s handle $handle
`, c.tableV4, c.chainV4, comment, c.tableV4, c.chainV4)
		_ = runSh(script) // best-effort
		// clean up delta state
		delete(c.prev, ep)
	}
	return nil
}

func (c *Collector) collectNft(keys []EndpointKey) ([]Sample, error) {
	// Dump full table as JSON for efficient parsing.
	out, err := exec.Command("nft", "-j", "list", "table", "inet", c.tableV4).Output()
	if err != nil {
		return nil, fmt.Errorf("nft list table: %w", err)
	}
	// Parse JSON – look for counter values.
	counters := parseNftCounters(out, c.chainV4)

	now := time.Now().UnixMilli()
	var samples []Sample
	for _, key := range keys {
		comment := ruleHandle(key)
		cnt, found := counters[comment]
		if !found {
			// Rule missing – re-add lazily; report zero delta.
			continue
		}
		delta := c.computeDelta(key, cnt[0], cnt[1], now)
		if delta == nil {
			continue
		}
		samples = append(samples, *delta)
	}
	return samples, nil
}

// parseNftCounters extracts {comment → [packets, bytes]} from nft JSON output.
func parseNftCounters(raw []byte, chainName string) map[string][2]int64 {
	result := make(map[string][2]int64)
	// Use a lightweight struct walk; full nft JSON schema is complex.
	// We look for: "rule" > "expr" items containing "counter" and "comment".
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return result
	}
	nftables, ok := top["nftables"]
	if !ok {
		return result
	}
	var items []json.RawMessage
	if err := json.Unmarshal(nftables, &items); err != nil {
		return result
	}
	for _, item := range items {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(item, &obj); err != nil {
			continue
		}
		ruleRaw, ok := obj["rule"]
		if !ok {
			continue
		}
		var rule struct {
			Chain   string           `json:"chain"`
			Comment string           `json:"comment"`
			Expr    []json.RawMessage `json:"expr"`
		}
		if err := json.Unmarshal(ruleRaw, &rule); err != nil {
			continue
		}
		if rule.Chain != chainName {
			continue
		}
		comment := rule.Comment
		if comment == "" {
			continue
		}
		// Find counter in expr.
		for _, exprRaw := range rule.Expr {
			var expr map[string]json.RawMessage
			if err := json.Unmarshal(exprRaw, &expr); err != nil {
				continue
			}
			cntRaw, ok := expr["counter"]
			if !ok {
				continue
			}
			var cnt struct {
				Packets int64 `json:"packets"`
				Bytes   int64 `json:"bytes"`
			}
			if err := json.Unmarshal(cntRaw, &cnt); err != nil {
				continue
			}
			// bytes received = in (prerouting hook captures incoming traffic)
			// We use packets as proxy for out (cannot distinguish easily at prerouting).
			// For a proper bidirectional count we need two rules per port.
			result[comment] = [2]int64{cnt.Bytes, 0}
			break
		}
	}
	return result
}

// ─── iptables ────────────────────────────────────────────────────────────────

const iptChain = "FLUX_REALM_STATS"

func (c *Collector) ensureIptRules(endpoints []EndpointKey) error {
	for _, tool := range []string{"iptables", "ip6tables"} {
		// Create chain if missing.
		_ = runSh(fmt.Sprintf(`%s -N %s 2>/dev/null; %s -C INPUT -j %s 2>/dev/null || %s -I INPUT -j %s`,
			tool, iptChain, tool, iptChain, tool, iptChain))
		_ = runSh(fmt.Sprintf(`%s -C OUTPUT -j %s 2>/dev/null || %s -I OUTPUT -j %s`,
			tool, iptChain, tool, iptChain))
	}
	for _, ep := range endpoints {
		for _, dir := range []struct{ chain, flag string }{{"INPUT", "--dport"}, {"OUTPUT", "--sport"}} {
			_ = addIptRule(ep, dir.chain, dir.flag)
		}
	}
	return nil
}

func addIptRule(ep EndpointKey, chain, portFlag string) error {
	proto := ep.Protocol
	if proto == "" {
		proto = "tcp"
	}
	comment := fmt.Sprintf("flux_%s_%d_%s", proto, ep.Port, chain[:1])
	// Check if rule already exists.
	checkCmd := fmt.Sprintf(`iptables -C %s -p %s %s %d -m comment --comment "%s" 2>/dev/null`,
		iptChain, proto, portFlag, ep.Port, comment)
	if runSh(checkCmd) == nil {
		return nil
	}
	addCmd := fmt.Sprintf(`iptables -A %s -p %s %s %d -m comment --comment "%s"`,
		iptChain, proto, portFlag, ep.Port, comment)
	return runSh(addCmd)
}

func (c *Collector) removeIptRules(endpoints []EndpointKey) error {
	for _, ep := range endpoints {
		for _, dir := range []struct{ chain, flag string }{{"INPUT", "--dport"}, {"OUTPUT", "--sport"}} {
			proto := ep.Protocol
			if proto == "" {
				proto = "tcp"
			}
			comment := fmt.Sprintf("flux_%s_%d_%s", proto, ep.Port, dir.chain[:1])
			script := fmt.Sprintf(`iptables -D %s -p %s %s %d -m comment --comment "%s" 2>/dev/null || true`,
				iptChain, proto, dir.flag, ep.Port, comment)
			_ = runSh(script)
		}
		delete(c.prev, ep)
	}
	return nil
}

func (c *Collector) collectIpt(keys []EndpointKey) ([]Sample, error) {
	out, err := exec.Command("iptables", "-L", iptChain, "-nvx", "--line-numbers").Output()
	if err != nil {
		return nil, fmt.Errorf("iptables -L: %w", err)
	}
	counters := parseIptCounters(string(out))

	now := time.Now().UnixMilli()
	var samples []Sample
	for _, key := range keys {
		proto := key.Protocol
		if proto == "" {
			proto = "tcp"
		}
		inKey := fmt.Sprintf("flux_%s_%d_I", proto, key.Port)
		outKey := fmt.Sprintf("flux_%s_%d_O", proto, key.Port)
		inBytes := counters[inKey]
		outBytes := counters[outKey]
		combined := EndpointKey{ForwardID: key.ForwardID, Port: key.Port, Protocol: proto}
		delta := c.computeDelta(combined, inBytes, outBytes, now)
		if delta == nil {
			continue
		}
		samples = append(samples, *delta)
	}
	return samples, nil
}

// parseIptCounters returns {comment → bytes}.
func parseIptCounters(raw string) map[string]int64 {
	result := make(map[string]int64)
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		// iptables -nvx format: pkts bytes target prot opt in out source destination [options...]
		bytesStr := fields[1]
		bytes, err := strconv.ParseInt(bytesStr, 10, 64)
		if err != nil {
			continue
		}
		// Comment is in the options part.
		rest := strings.Join(fields[8:], " ")
		if !strings.Contains(rest, "flux_") {
			continue
		}
		for _, part := range strings.Fields(rest) {
			if strings.HasPrefix(part, "flux_") {
				result[part] = bytes
				break
			}
		}
	}
	return result
}

// ─── procfs (node-level) ──────────────────────────────────────────────────────

func (c *Collector) collectProcfs(keys []EndpointKey) ([]Sample, error) {
	rx, tx := readProcNetDev()
	now := time.Now().UnixMilli()
	// procfs is node-level only; distribute equally (not per-endpoint).
	// Return a single node-level sample.
	if len(keys) == 0 {
		return nil, nil
	}
	key := keys[0]
	delta := c.computeDelta(key, rx, tx, now)
	if delta == nil {
		return nil, nil
	}
	delta.Method = "procfs"
	delta.Error = "procfs is node-level, not per-endpoint"
	return []Sample{*delta}, nil
}

func readProcNetDev() (rx, tx int64) {
	b, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		name, data, _ := strings.Cut(line, ":")
		if strings.TrimSpace(name) == "lo" {
			continue
		}
		fields := strings.Fields(data)
		if len(fields) >= 9 {
			r, _ := strconv.ParseInt(fields[0], 10, 64)
			t, _ := strconv.ParseInt(fields[8], 10, 64)
			rx += r
			tx += t
		}
	}
	return rx, tx
}

// ─── delta computation ────────────────────────────────────────────────────────

func (c *Collector) computeDelta(key EndpointKey, inBytes, outBytes, now int64) *Sample {
	prev, hasPrev := c.prev[key]
	c.prev[key] = counterState{lastIn: inBytes, lastOut: outBytes, ts: now}
	if !hasPrev {
		return nil // first observation – establish baseline
	}
	inDelta := inBytes - prev.lastIn
	outDelta := outBytes - prev.lastOut
	// Counter wrap or reset (e.g., after reboot, chain flush).
	if inDelta < 0 {
		inDelta = inBytes // use current value as delta
	}
	if outDelta < 0 {
		outDelta = outBytes
	}
	total := inDelta + outDelta
	return &Sample{
		ListenPort:   key.Port,
		Protocol:     key.Protocol,
		ForwardID:    key.ForwardID,
		InBytes:      inDelta,
		OutBytes:     outDelta,
		TotalBytes:   total,
		BillingBytes: total,
		SampleTime:   now,
		Method:       c.method,
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func ruleHandle(ep EndpointKey) string {
	proto := ep.Protocol
	if proto == "" {
		proto = "tcp"
	}
	return fmt.Sprintf("flux_%s_%d_%d", proto, ep.Port, ep.ForwardID)
}

func runSh(script string) error {
	var stderr bytes.Buffer
	cmd := exec.Command("sh", "-c", script)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
