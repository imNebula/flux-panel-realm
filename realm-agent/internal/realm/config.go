package realm

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Config struct {
	Log       *LogConfig       `json:"log,omitempty"`
	DNS       *DNSConfig       `json:"dns,omitempty"`
	Network   *NetworkConfig   `json:"network,omitempty"`
	Endpoints []EndpointConfig `json:"endpoints"`
}

type LogConfig struct {
	Level  string `json:"level,omitempty"`
	Output string `json:"output,omitempty"`
}

type DNSConfig struct {
	Mode        string   `json:"mode,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
	Nameservers []string `json:"nameservers,omitempty"`
	MinTTL      uint32   `json:"min_ttl,omitempty"`
	MaxTTL      uint32   `json:"max_ttl,omitempty"`
	CacheSize   uint32   `json:"cache_size,omitempty"`
}

type NetworkConfig struct {
	NoTCP              *bool   `json:"no_tcp,omitempty"`
	UseUDP             *bool   `json:"use_udp,omitempty"`
	IPv6Only           *bool   `json:"ipv6_only,omitempty"`
	TCPTimeout         *uint32 `json:"tcp_timeout,omitempty"`
	UDPTimeout         *uint32 `json:"udp_timeout,omitempty"`
	TCPKeepalive       *uint32 `json:"tcp_keepalive,omitempty"`
	TCPKeepaliveProbe  *uint32 `json:"tcp_keepalive_probe,omitempty"`
	SendMPTCP          *bool   `json:"send_mptcp,omitempty"`
	AcceptMPTCP        *bool   `json:"accept_mptcp,omitempty"`
	SendProxy          *bool   `json:"send_proxy,omitempty"`
	SendProxyVersion   *uint8  `json:"send_proxy_version,omitempty"`
	AcceptProxy        *bool   `json:"accept_proxy,omitempty"`
	AcceptProxyTimeout *uint32 `json:"accept_proxy_timeout,omitempty"`
}

type EndpointConfig struct {
	Name            string         `json:"-"`
	ForwardID       int64          `json:"-"`
	TunnelID        int64          `json:"-"`
	UserID          int64          `json:"-"`
	Listen          string         `json:"listen"`
	Remote          string         `json:"remote"`
	ExtraRemotes    []string       `json:"extra_remotes,omitempty"`
	Balance         string         `json:"balance,omitempty"`
	Through         string         `json:"through,omitempty"`
	Interface       string         `json:"interface,omitempty"`
	ListenInterface string         `json:"listen_interface,omitempty"`
	ListenTransport string         `json:"listen_transport,omitempty"`
	RemoteTransport string         `json:"remote_transport,omitempty"`
	Network         *NetworkConfig `json:"network,omitempty"`
	Disabled        bool           `json:"-"`
	Unsupported     string         `json:"-"`
}

func (e *EndpointConfig) UnmarshalJSON(data []byte) error {
	type endpointConfigAlias EndpointConfig
	aux := struct {
		Name      string `json:"name"`
		ForwardID int64  `json:"forward_id"`
		TunnelID  int64  `json:"tunnel_id"`
		UserID    int64  `json:"user_id"`
		*endpointConfigAlias
	}{
		endpointConfigAlias: (*endpointConfigAlias)(e),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	e.Name = aux.Name
	e.ForwardID = aux.ForwardID
	e.TunnelID = aux.TunnelID
	e.UserID = aux.UserID
	return nil
}

type LegacyService struct {
	Name      string                 `json:"name"`
	Addr      string                 `json:"addr"`
	Metadata  map[string]any         `json:"metadata"`
	Handler   map[string]any         `json:"handler"`
	Listener  map[string]any         `json:"listener"`
	Forwarder map[string]any         `json:"forwarder"`
	Limiter   string                 `json:"limiter"`
	Raw       map[string]interface{} `json:"-"`
}

type LegacyChain struct {
	Name      string
	Remote    string
	Protocol  string
	Interface string
}

func DefaultConfig(logPath string, endpoints []EndpointConfig) Config {
	useUDP := hasUDP(endpoints)
	return Config{
		Log: &LogConfig{Level: "info", Output: logPath},
		Network: &NetworkConfig{
			UseUDP: &useUDP,
		},
		Endpoints: endpoints,
	}
}

func Validate(cfg Config) error {
	if len(cfg.Endpoints) == 0 {
		return nil
	}
	for _, ep := range cfg.Endpoints {
		if ep.Disabled {
			continue
		}
		if err := validateAddress(ep.Listen, true); err != nil {
			return fmt.Errorf("endpoint %s listen: %w", ep.Name, err)
		}
		if err := validateAddress(ep.Remote, false); err != nil {
			return fmt.Errorf("endpoint %s remote: %w", ep.Name, err)
		}
		for _, remote := range ep.ExtraRemotes {
			if err := validateAddress(remote, false); err != nil {
				return fmt.Errorf("endpoint %s extra remote: %w", ep.Name, err)
			}
		}
		if err := validateTransport(ep.ListenTransport); err != nil {
			return fmt.Errorf("endpoint %s listen_transport: %w", ep.Name, err)
		}
		if err := validateTransport(ep.RemoteTransport); err != nil {
			return fmt.Errorf("endpoint %s remote_transport: %w", ep.Name, err)
		}
		if ep.Balance != "" && !strings.HasPrefix(ep.Balance, "roundrobin") && !strings.HasPrefix(ep.Balance, "iphash") {
			return fmt.Errorf("endpoint %s balance must be roundrobin or iphash", ep.Name)
		}
	}
	_, err := json.Marshal(cfg)
	return err
}

func LegacyServiceToEndpoint(s LegacyService, chains map[string]LegacyChain) (EndpointConfig, bool, error) {
	if s.Name == "" || s.Addr == "" {
		return EndpointConfig{}, false, fmt.Errorf("legacy service requires name and addr")
	}
	if strings.Contains(s.Name, "_unsupported") {
		return EndpointConfig{Name: s.Name, Disabled: true, Unsupported: "legacy service marked unsupported"}, false, nil
	}
	protocol := lowerStringFromMap(s.Listener, "type")
	if protocol == "" {
		protocol = lowerStringFromMap(s.Handler, "type")
	}
	if protocol == "udp" {
		noTCP, useUDP := true, true
		return fillEndpointNetwork(mapService(s, chains), &NetworkConfig{NoTCP: &noTCP, UseUDP: &useUDP}), true, nil
	}
	if protocol != "" && protocol != "tcp" && protocol != "tls" && protocol != "ws" && protocol != "wss" {
		ep := EndpointConfig{Name: s.Name, Disabled: true, Unsupported: "Realm cannot emulate Gost protocol " + protocol}
		return ep, false, nil
	}
	ep := mapService(s, chains)
	return ep, false, nil
}

func mapService(s LegacyService, chains map[string]LegacyChain) EndpointConfig {
	ep := EndpointConfig{Name: s.Name, Listen: normalizeListen(s.Addr)}
	listenerProtocol := lowerStringFromMap(s.Listener, "type")
	if listenerProtocol == "tls" || listenerProtocol == "ws" || listenerProtocol == "wss" {
		ep.ListenTransport = listenerProtocol
	}
	if iface, _ := s.Metadata["interface"].(string); iface != "" {
		ep.ListenInterface = iface
	}
	if chainName := stringFromMap(s.Handler, "chain"); chainName != "" {
		ch := chains[chainName]
		ep.Remote = ch.Remote
		ep.RemoteTransport = gostProtocolToRealmTransport(ch.Protocol)
		ep.Interface = ch.Interface
		return ep
	}
	remote, extras, balance := parseForwarder(s.Forwarder)
	ep.Remote = remote
	ep.ExtraRemotes = extras
	ep.Balance = balance
	return ep
}

func fillEndpointNetwork(ep EndpointConfig, network *NetworkConfig) EndpointConfig {
	ep.Network = network
	return ep
}

func parseForwarder(forwarder map[string]any) (string, []string, string) {
	nodes, _ := forwarder["nodes"].([]any)
	var remotes []string
	for _, n := range nodes {
		if m, ok := n.(map[string]any); ok {
			if addr, _ := m["addr"].(string); addr != "" {
				remotes = append(remotes, addr)
			}
		}
	}
	var balance string
	if selector, ok := forwarder["selector"].(map[string]any); ok {
		switch strings.ToLower(fmt.Sprint(selector["strategy"])) {
		case "round", "roundrobin":
			balance = "roundrobin"
		case "hash", "iphash":
			balance = "iphash"
		}
	}
	if len(remotes) == 0 {
		return "127.0.0.1:9", nil, balance
	}
	return remotes[0], remotes[1:], balance
}

func ParseLegacyChain(data map[string]any) LegacyChain {
	ch := LegacyChain{Name: fmt.Sprint(data["name"])}
	hops, _ := data["hops"].([]any)
	if len(hops) == 0 {
		return ch
	}
	hop, _ := hops[0].(map[string]any)
	nodes, _ := hop["nodes"].([]any)
	if len(nodes) == 0 {
		return ch
	}
	node, _ := nodes[0].(map[string]any)
	ch.Remote, _ = node["addr"].(string)
	ch.Interface, _ = node["interface"].(string)
	if dialer, ok := node["dialer"].(map[string]any); ok {
		ch.Protocol = lowerStringFromMap(dialer, "type")
	}
	return ch
}

func gostProtocolToRealmTransport(protocol string) string {
	switch strings.ToLower(protocol) {
	case "tls", "ws", "wss":
		return strings.ToLower(protocol)
	default:
		return ""
	}
}

func validateAddress(addr string, listen bool) error {
	if addr == "" {
		return fmt.Errorf("address is empty")
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if listen && host == "" {
		return nil
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("invalid port %q", port)
	}
	if host == "" {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil {
		return nil
	}
	if strings.ContainsAny(host, " /\\") {
		return fmt.Errorf("invalid host %q", host)
	}
	return nil
}

func validateTransport(v string) error {
	if v == "" {
		return nil
	}
	name := strings.ToLower(strings.Split(v, ";")[0])
	switch name {
	case "tls", "ws", "wss":
		return nil
	default:
		return fmt.Errorf("unsupported transport %q", name)
	}
}

func normalizeListen(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "0.0.0.0" + addr
	}
	return addr
}

func lowerStringFromMap(m map[string]any, k string) string {
	return strings.ToLower(stringFromMap(m, k))
}

func stringFromMap(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func hasUDP(eps []EndpointConfig) bool {
	for _, ep := range eps {
		if ep.Network != nil && ep.Network.UseUDP != nil && *ep.Network.UseUDP {
			return true
		}
	}
	return false
}
