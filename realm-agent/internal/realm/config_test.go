package realm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLegacyPortForwardToRealmEndpoint(t *testing.T) {
	svc := LegacyService{
		Name:     "10_1_3_tcp",
		Addr:     "0.0.0.0:18080",
		Listener: map[string]any{"type": "tcp"},
		Handler:  map[string]any{"type": "tcp"},
		Forwarder: map[string]any{
			"nodes": []any{
				map[string]any{"addr": "example.com:80"},
				map[string]any{"addr": "example.net:80"},
			},
			"selector": map[string]any{"strategy": "roundrobin"},
		},
	}
	ep, udp, err := LegacyServiceToEndpoint(svc, nil)
	if err != nil {
		t.Fatal(err)
	}
	if udp {
		t.Fatal("tcp service was treated as udp")
	}
	if ep.Listen != "0.0.0.0:18080" || ep.Remote != "example.com:80" {
		t.Fatalf("unexpected endpoint: %#v", ep)
	}
	if len(ep.ExtraRemotes) != 1 || ep.ExtraRemotes[0] != "example.net:80" {
		t.Fatalf("extra remotes not mapped: %#v", ep.ExtraRemotes)
	}
	if ep.Balance != "roundrobin" {
		t.Fatalf("balance not mapped: %q", ep.Balance)
	}
	if err := Validate(DefaultConfig("stdout", []EndpointConfig{ep})); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyTunnelChainToRealmTransport(t *testing.T) {
	chains := map[string]LegacyChain{
		"hop_1_chains": {Name: "hop_1_chains", Remote: "203.0.113.10:2443", Protocol: "tls", Interface: "eth0"},
	}
	svc := LegacyService{
		Name:     "1_2_3_tcp",
		Addr:     ":18081",
		Listener: map[string]any{"type": "tcp"},
		Handler:  map[string]any{"type": "tcp", "chain": "hop_1_chains"},
	}
	ep, _, err := LegacyServiceToEndpoint(svc, chains)
	if err != nil {
		t.Fatal(err)
	}
	if ep.Listen != "0.0.0.0:18081" || ep.Remote != "203.0.113.10:2443" {
		t.Fatalf("unexpected endpoint: %#v", ep)
	}
	if ep.RemoteTransport != "tls" || ep.Interface != "eth0" {
		t.Fatalf("chain transport/interface not mapped: %#v", ep)
	}
}

func TestUnsupportedGostProtocolIsMarkedUnsupported(t *testing.T) {
	svc := LegacyService{
		Name:     "legacy_quic",
		Addr:     ":18082",
		Listener: map[string]any{"type": "quic"},
		Handler:  map[string]any{"type": "quic"},
	}
	ep, _, err := LegacyServiceToEndpoint(svc, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ep.Disabled || ep.Unsupported == "" {
		t.Fatalf("expected unsupported endpoint, got %#v", ep)
	}
}

func TestEndpointMetadataIsAcceptedButNotWrittenToRealmConfig(t *testing.T) {
	raw := []byte(`{
		"name": "forward-10-in",
		"forward_id": 10,
		"tunnel_id": 20,
		"user_id": 30,
		"listen": "0.0.0.0:18080",
		"remote": "example.com:80"
	}`)
	var ep EndpointConfig
	if err := json.Unmarshal(raw, &ep); err != nil {
		t.Fatal(err)
	}
	if ep.Name != "forward-10-in" || ep.ForwardID != 10 || ep.TunnelID != 20 || ep.UserID != 30 {
		t.Fatalf("metadata was not preserved for agent internals: %#v", ep)
	}
	out, err := json.Marshal(DefaultConfig("stdout", []EndpointConfig{ep}))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) == "" || containsAny(string(out), []string{"forward_id", "tunnel_id", "user_id", "forward-10-in"}) {
		t.Fatalf("internal metadata leaked into realm config: %s", out)
	}
}

func containsAny(s string, parts []string) bool {
	for _, part := range parts {
		if strings.Contains(s, part) {
			return true
		}
	}
	return false
}
