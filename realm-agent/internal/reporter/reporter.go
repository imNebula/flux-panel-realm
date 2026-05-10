package reporter

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/imnebula/flux-panel-realm/realm-agent/internal/crypto"
	"github.com/imnebula/flux-panel-realm/realm-agent/internal/platform"
	"github.com/imnebula/flux-panel-realm/realm-agent/internal/realm"
	"github.com/imnebula/flux-panel-realm/realm-agent/internal/traffic"
)

type Config struct {
	ServerAddr       string
	Secret           string
	AgentVersion     string
	AgentProcessName string
	AgentName        string
	ServiceName      string
	InstanceName     string
	ConfigDir        string
	RealmBinaryPath  string
	RealmProcessName string
	Mode             string
}

type Reporter struct {
	cfg        Config
	env        platform.Environment
	manager    *realm.Manager
	aes        *crypto.AESCrypto
	conn       *websocket.Conn
	mu         sync.Mutex
	collector  *traffic.Collector
	// lastTrafficKeys tracks active endpoints for traffic collection.
	lastKeys   []traffic.EndpointKey
	// latencyCfg is the per-agent latency sampling configuration.
	latencyCfg traffic.LatencyConfig
}

type commandMessage struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	RequestID string          `json:"requestId,omitempty"`
}

type commandResponse struct {
	Type      string `json:"type"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

func New(cfg Config, env platform.Environment, manager *realm.Manager) *Reporter {
	aes, _ := crypto.New(cfg.Secret)
	col := traffic.NewCollector(env.TrafficStatsMethod, env.TrafficStatsReason)
	return &Reporter{
		cfg:        cfg,
		env:        env,
		manager:    manager,
		aes:        aes,
		collector:  col,
		latencyCfg: traffic.DefaultLatencyConfig(),
	}
}

func (r *Reporter) Run(ctx context.Context) error {
	for ctx.Err() == nil {
		if err := r.connect(ctx); err != nil {
			fmt.Printf("connect failed: %v\n", err)
			sleep(ctx, 5*time.Second)
			continue
		}
		r.handle(ctx)
		sleep(ctx, 3*time.Second)
	}
	return ctx.Err()
}

func (r *Reporter) connect(ctx context.Context) error {
	u := r.wsURL()
	d := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := d.DialContext(ctx, u, nil)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.conn = conn
	r.mu.Unlock()
	return nil
}

func (r *Reporter) handle(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, msg, err := r.conn.ReadMessage()
			if err != nil {
				return
			}
			r.route(msg)
		}
	}()
	heartbeatTicker := time.NewTicker(2 * time.Second)
	defer heartbeatTicker.Stop()
	trafficTicker := time.NewTicker(60 * time.Second)
	defer trafficTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = r.conn.Close()
			return
		case <-done:
			return
		case <-heartbeatTicker.C:
			_ = r.sendSystemInfo()
		case <-trafficTicker.C:
			r.collectAndSendTraffic()
		}
	}
}

func (r *Reporter) route(msg []byte) {
	plain := r.decryptIfNeeded(msg)
	var wrapper struct {
		Type       string          `json:"type"`
		Compressed bool            `json:"compressed"`
		Data       json.RawMessage `json:"data"`
		RequestID  string          `json:"requestId"`
	}
	if err := json.Unmarshal(plain, &wrapper); err == nil && wrapper.Compressed {
		if b, err := gunzip(wrapper.Data); err == nil {
			plain = b
		}
	}
	var cmd commandMessage
	if err := json.Unmarshal(plain, &cmd); err != nil {
		r.send(commandResponse{Type: "ParseError", Success: false, Message: err.Error()})
		return
	}
	if cmd.Type == "call" || cmd.Type == "" {
		return
	}
	resp := r.handleCommand(cmd)
	resp.RequestID = cmd.RequestID
	r.send(resp)
}

func (r *Reporter) handleCommand(cmd commandMessage) commandResponse {
	switch cmd.Type {
	case "AddService", "UpdateService":
		var services []realm.LegacyService
		if err := json.Unmarshal(cmd.Data, &services); err != nil {
			return errResp(cmd.Type+"Response", err)
		}
		res := r.manager.AddOrUpdateServices(services)
		return applyResp(cmd.Type+"Response", res)
	case "DeleteService", "PauseService", "ResumeService":
		names, err := namesFromPayload(cmd.Data)
		if err != nil {
			return errResp(cmd.Type+"Response", err)
		}
		var res realm.ApplyResult
		switch cmd.Type {
		case "DeleteService":
			res = r.manager.DeleteServices(names)
		case "PauseService":
			res = r.manager.PauseServices(names)
		case "ResumeService":
			res = r.manager.ResumeServices(names)
		}
		return applyResp(cmd.Type+"Response", res)
	case "AddChains":
		var raw map[string]any
		if err := json.Unmarshal(cmd.Data, &raw); err != nil {
			return errResp("AddChainsResponse", err)
		}
		res := r.manager.AddOrUpdateChain(realm.ParseLegacyChain(raw))
		return applyResp("AddChainsResponse", res)
	case "UpdateChains":
		var raw struct {
			Chain string         `json:"chain"`
			Data  map[string]any `json:"data"`
		}
		if err := json.Unmarshal(cmd.Data, &raw); err != nil {
			return errResp("UpdateChainsResponse", err)
		}
		res := r.manager.AddOrUpdateChain(realm.ParseLegacyChain(raw.Data))
		return applyResp("UpdateChainsResponse", res)
	case "DeleteChains":
		var raw struct {
			Chain string `json:"chain"`
		}
		if err := json.Unmarshal(cmd.Data, &raw); err != nil {
			return errResp("DeleteChainsResponse", err)
		}
		res := r.manager.DeleteChain(raw.Chain)
		return applyResp("DeleteChainsResponse", res)
	case "ApplyRealmConfig":
		var cfg realm.Config
		if err := json.Unmarshal(cmd.Data, &cfg); err != nil {
			return errResp("ApplyRealmConfigResponse", err)
		}
		res := r.manager.ApplyConfig(cfg, []string{"full-config"})
		return applyResp("ApplyRealmConfigResponse", res)
	case "TcpPing":
		data, err := r.tcpPing(cmd.Data)
		if err != nil {
			return errResp("TcpPingResponse", err)
		}
		return commandResponse{Type: "TcpPingResponse", Success: true, Message: "OK", Data: data}
	case "SetProtocol", "AddLimiters", "UpdateLimiters", "DeleteLimiters":
		return commandResponse{Type: cmd.Type + "Response", Success: true, Message: "OK", Data: map[string]any{"deprecated": true, "note": "Realm agent accepted legacy command as compatibility no-op"}}
	case "LatencyProbe":
		var req traffic.ProbeRequest
		if err := json.Unmarshal(cmd.Data, &req); err != nil {
			return errResp("LatencyProbeResponse", err)
		}
		sample := traffic.RunProbe(context.Background(), req, 0)
		return commandResponse{Type: "LatencyProbeResponse", Success: sample.Success == 1, Message: "OK", Data: sample}
	case "UpdateLatencyConfig":
		var cfg traffic.LatencyConfig
		if err := json.Unmarshal(cmd.Data, &cfg); err != nil {
			return errResp("UpdateLatencyConfigResponse", err)
		}
		r.latencyCfg = cfg
		return commandResponse{Type: "UpdateLatencyConfigResponse", Success: true, Message: "OK"}
	case "CollectTrafficNow":
		r.collectAndSendTraffic()
		return commandResponse{Type: "CollectTrafficNowResponse", Success: true, Message: "OK"}
	default:
		return commandResponse{Type: "UnknownCommandResponse", Success: false, Message: "unknown command: " + cmd.Type}
	}
}

func (r *Reporter) sendSystemInfo() error {
	rx, tx := netBytes()
	info := map[string]any{
		"uptime":                      uptime(),
		"bytes_received":              rx,
		"bytes_transmitted":           tx,
		"cpu_usage":                   0,
		"memory_usage":                memoryUsage(),
		"agent_version":               r.cfg.AgentVersion,
		"realm_version":               r.manager.Version(),
		"realm_binary_path":           r.cfg.RealmBinaryPath,
		"realm_config_dir":            r.cfg.ConfigDir,
		"realm_process_name":          r.cfg.RealmProcessName,
		"realm_service_name":          r.cfg.ServiceName,
		"agent_process_name":          r.cfg.AgentProcessName,
		"instance_name":               r.cfg.InstanceName,
		"os":                          r.env.OS,
		"distro":                      r.env.Distro,
		"os_version":                  r.env.Version,
		"arch":                        r.env.Arch,
		"libc":                        r.env.Libc,
		"init_system":                 r.env.InitSystem,
		"container_type":              r.env.ContainerType,
		"capabilities":                r.capabilities(),
		"currently_running_processes": r.manager.Processes(),
		"config_hash":                 r.manager.ConfigHash(),
		"endpoint_count":              r.manager.EndpointCount(),
		"active_forward_count":        r.manager.EndpointCount(),
		"active_tunnel_count":         0,
		"last_apply_time":             r.manager.LastApply().ApplyID,
		"last_apply_status":           r.manager.LastApply().Success,
		"last_apply_error":            r.manager.LastApply().ErrorMessage,
		"last_apply":                  r.manager.LastApply(),
	}
	b, _ := json.Marshal(info)
	return r.writeEncrypted(b)
}

// collectAndSendTraffic gathers per-endpoint traffic samples and sends them
// to the backend via a "TrafficSamples" message.
func (r *Reporter) collectAndSendTraffic() {
	if r.collector == nil || len(r.lastKeys) == 0 {
		return
	}
	samples, err := r.collector.Collect(r.lastKeys)
	if err != nil || len(samples) == 0 {
		return
	}
	msg := map[string]any{
		"type":    "TrafficSamples",
		"samples": samples,
	}
	b, _ := json.Marshal(msg)
	_ = r.writeEncrypted(b)
}

// updateTrafficEndpoints updates the active endpoint keys after a config apply.
func (r *Reporter) updateTrafficEndpoints(endpoints []traffic.EndpointKey) {
	if r.collector == nil {
		return
	}
	// Remove rules for deleted endpoints.
	deleted := diffDeleted(r.lastKeys, endpoints)
	if len(deleted) > 0 {
		_ = r.collector.RemoveRules(deleted)
	}
	// Ensure rules for new endpoints.
	if len(endpoints) > 0 {
		_ = r.collector.EnsureRules(endpoints)
	}
	r.lastKeys = endpoints
}

func diffDeleted(prev, next []traffic.EndpointKey) []traffic.EndpointKey {
	nextSet := make(map[traffic.EndpointKey]bool)
	for _, k := range next {
		nextSet[k] = true
	}
	var deleted []traffic.EndpointKey
	for _, k := range prev {
		if !nextSet[k] {
			deleted = append(deleted, k)
		}
	}
	return deleted
}

func (r *Reporter) capabilities() map[string]any {
	return map[string]any{
		"tcp":                  true,
		"udp":                  true,
		"tcp_udp_same_port":    true,
		"ws":                   true,
		"wss":                  true,
		"tls":                  true,
		"proxy_protocol":       true,
		"balance":              true,
		"extra_remotes":        true,
		"mptcp":                r.env.OS == "linux",
		"traffic_stats_method": r.env.TrafficStatsMethod,
		"traffic_stats_reason": r.env.TrafficStatsReason,
		"latency_probe_tcp":    true,
		"latency_probe_udp":    false,
		"realtime_apply":       true,
		"custom_process_name":  true,
		"multi_process":        true,
	}
}

func (r *Reporter) send(resp commandResponse) {
	b, _ := json.Marshal(resp)
	_ = r.writeEncrypted(b)
}

func (r *Reporter) writeEncrypted(plain []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.conn == nil {
		return fmt.Errorf("not connected")
	}
	msg := plain
	if r.aes != nil {
		if encrypted, err := r.aes.Encrypt(plain); err == nil {
			msg, _ = json.Marshal(map[string]any{
				"encrypted": true,
				"data":      encrypted,
				"timestamp": time.Now().Unix(),
			})
		}
	}
	_ = r.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return r.conn.WriteMessage(websocket.TextMessage, msg)
}

func (r *Reporter) decryptIfNeeded(msg []byte) []byte {
	var wrapped struct {
		Encrypted bool   `json:"encrypted"`
		Data      string `json:"data"`
	}
	if err := json.Unmarshal(msg, &wrapped); err == nil && wrapped.Encrypted && wrapped.Data != "" && r.aes != nil {
		if plain, err := r.aes.Decrypt(wrapped.Data); err == nil {
			return plain
		}
	}
	return msg
}

func (r *Reporter) wsURL() string {
	addr := r.cfg.ServerAddr
	addr = strings.TrimPrefix(addr, "http://")
	addr = strings.TrimPrefix(addr, "https://")
	q := url.Values{}
	q.Set("type", "1")
	q.Set("secret", r.cfg.Secret)
	q.Set("version", r.cfg.AgentVersion)
	q.Set("http", "0")
	q.Set("tls", "1")
	q.Set("socks", "0")
	return "ws://" + addr + "/system-info?" + q.Encode()
}

func namesFromPayload(data []byte) ([]string, error) {
	var obj struct {
		Services []string `json:"services"`
	}
	if err := json.Unmarshal(data, &obj); err == nil && len(obj.Services) > 0 {
		return obj.Services, nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		return arr, nil
	}
	return nil, fmt.Errorf("payload must include services")
}

func applyResp(typ string, res realm.ApplyResult) commandResponse {
	if !res.Success {
		return commandResponse{Type: typ, Success: false, Message: res.ErrorMessage, Data: res}
	}
	return commandResponse{Type: typ, Success: true, Message: "OK", Data: res}
}

func errResp(typ string, err error) commandResponse {
	return commandResponse{Type: typ, Success: false, Message: err.Error()}
}

func (r *Reporter) tcpPing(data []byte) (map[string]any, error) {
	var req struct {
		IP      string `json:"ip"`
		Port    int    `json:"port"`
		Count   int    `json:"count"`
		Timeout int    `json:"timeout"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	if req.Count <= 0 || req.Count > 10 {
		req.Count = 2
	}
	if req.Timeout <= 0 || req.Timeout > 30000 {
		req.Timeout = 3000
	}
	target := net.JoinHostPort(req.IP, strconv.Itoa(req.Port))
	var total time.Duration
	fail := 0
	for i := 0; i < req.Count; i++ {
		start := time.Now()
		c, err := net.DialTimeout("tcp", target, time.Duration(req.Timeout)*time.Millisecond)
		if err != nil {
			fail++
			continue
		}
		total += time.Since(start)
		_ = c.Close()
	}
	successes := req.Count - fail
	avg := 0.0
	if successes > 0 {
		avg = float64(total.Milliseconds()) / float64(successes)
	}
	return map[string]any{
		"ip":           req.IP,
		"port":         req.Port,
		"success":      successes > 0,
		"averageTime":  avg,
		"packetLoss":   float64(fail) * 100 / float64(req.Count),
		"errorMessage": "",
	}, nil
}

func gunzip(raw []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	return io.ReadAll(gr)
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func uptime() uint64 {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	f, _ := strconv.ParseFloat(strings.Fields(string(b))[0], 64)
	return uint64(f)
}

func memoryUsage() float64 {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	val := map[string]float64{}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			v, _ := strconv.ParseFloat(fields[1], 64)
			val[strings.TrimSuffix(fields[0], ":")] = v
		}
	}
	if val["MemTotal"] == 0 {
		return 0
	}
	available := val["MemAvailable"]
	if available == 0 {
		available = val["MemFree"]
	}
	return (val["MemTotal"] - available) * 100 / val["MemTotal"]
}

func netBytes() (uint64, uint64) {
	b, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	var rx, tx uint64
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		name, data, _ := strings.Cut(line, ":")
		if strings.TrimSpace(name) == "lo" {
			continue
		}
		fields := strings.Fields(data)
		if len(fields) >= 16 {
			r, _ := strconv.ParseUint(fields[0], 10, 64)
			t, _ := strconv.ParseUint(fields[8], 10, 64)
			rx += r
			tx += t
		}
	}
	return rx, tx
}
