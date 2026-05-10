// Package traffic - latency sampler for per-endpoint TCP connect probing.
// Performs low-frequency latency sampling only when an endpoint has seen
// recent traffic (traffic-triggered sampling), avoiding idle port probing.
package traffic

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strconv"
	"time"
)

// LatencyConfig holds configuration for latency sampling.
type LatencyConfig struct {
	Enabled       bool   `json:"latency_sample_enabled"`
	IntervalSec   int    `json:"sample_interval"`
	TimeoutMs     int    `json:"sample_timeout"`
	Count         int    `json:"sample_count"`
	RetentionDays int    `json:"retention_days"`
	ProbeMode     string `json:"probe_mode"`   // "tcp_connect" | "http_head"
	ProbeTarget   string `json:"probe_target"` // override target (e.g. "1.1.1.1:80")
}

// DefaultLatencyConfig returns safe defaults (disabled, low frequency).
func DefaultLatencyConfig() LatencyConfig {
	return LatencyConfig{
		Enabled:       false,
		IntervalSec:   120,
		TimeoutMs:     3000,
		Count:         2,
		RetentionDays: 7,
		ProbeMode:     "tcp_connect",
	}
}

// LatencySample is a single probe result returned to the backend.
type LatencySample struct {
	NodeID    int64   `json:"node_id"`
	TunnelID  int64   `json:"tunnel_id,omitempty"`
	ForwardID int64   `json:"forward_id,omitempty"`
	Protocol  string  `json:"protocol"`
	ProbeMode string  `json:"probe_mode"`
	Target    string  `json:"target"`
	Success   int     `json:"success"` // 1=ok 0=fail
	LatencyMs float64 `json:"latency_ms"`
	JitterMs  float64 `json:"jitter_ms"`
	Error     string  `json:"error,omitempty"`
	SampledAt int64   `json:"sampled_at"`
}

// ProbeRequest is sent by the backend to trigger a latency probe.
type ProbeRequest struct {
	Target    string `json:"target"`     // host:port
	Count     int    `json:"count"`
	TimeoutMs int    `json:"timeout"`
	ForwardID int64  `json:"forward_id,omitempty"`
	TunnelID  int64  `json:"tunnel_id,omitempty"`
}

// ProbeResult is the response from a latency probe.
func RunProbe(ctx context.Context, req ProbeRequest, nodeID int64) LatencySample {
	count := req.Count
	if count <= 0 || count > 10 {
		count = 2
	}
	timeoutMs := req.TimeoutMs
	if timeoutMs <= 0 || timeoutMs > 30000 {
		timeoutMs = 3000
	}
	timeout := time.Duration(timeoutMs) * time.Millisecond
	target := req.Target
	if target == "" {
		return LatencySample{
			NodeID:    nodeID,
			ForwardID: req.ForwardID,
			TunnelID:  req.TunnelID,
			Protocol:  "tcp",
			ProbeMode: "tcp_connect",
			Target:    target,
			Success:   0,
			Error:     "empty target",
			SampledAt: time.Now().UnixMilli(),
		}
	}

	var latencies []float64
	var failed int
	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			failed = count - i
			break
		default:
		}
		start := time.Now()
		conn, err := net.DialTimeout("tcp", target, timeout)
		elapsed := time.Since(start).Seconds() * 1000 // ms
		if err != nil {
			failed++
			continue
		}
		_ = conn.Close()
		latencies = append(latencies, elapsed)
	}

	sample := LatencySample{
		NodeID:    nodeID,
		ForwardID: req.ForwardID,
		TunnelID:  req.TunnelID,
		Protocol:  "tcp",
		ProbeMode: "tcp_connect",
		Target:    target,
		SampledAt: time.Now().UnixMilli(),
	}
	if len(latencies) == 0 {
		sample.Success = 0
		sample.Error = fmt.Sprintf("all %d probes failed", count)
		return sample
	}
	sample.Success = 1
	avg := sum(latencies) / float64(len(latencies))
	sample.LatencyMs = avg
	sample.JitterMs = jitter(latencies)
	if failed > 0 {
		// partial success – encode packet loss info
		sample.Error = fmt.Sprintf("%.0f%% packet loss", float64(failed)*100/float64(count))
	}
	return sample
}

// ─── aggregate helpers ─────────────────────────────────────────────────────

// AggregateStats computes statistical aggregates from a list of latency samples.
type AggregateStats struct {
	AvgMs       float64 `json:"avg_ms"`
	MinMs       float64 `json:"min_ms"`
	MaxMs       float64 `json:"max_ms"`
	P50Ms       float64 `json:"p50_ms"`
	P95Ms       float64 `json:"p95_ms"`
	P99Ms       float64 `json:"p99_ms"`
	LossRate    float64 `json:"loss_rate"`
	SampleCount int     `json:"sample_count"`
}

// Aggregate computes statistics from raw latency/success data.
// successes: list of successful latency values (ms)
// total: total number of probes attempted (including failures)
func Aggregate(successes []float64, total int) AggregateStats {
	if total == 0 {
		return AggregateStats{}
	}
	lossRate := 1.0 - float64(len(successes))/float64(total)
	if len(successes) == 0 {
		return AggregateStats{LossRate: lossRate, SampleCount: total}
	}
	sorted := make([]float64, len(successes))
	copy(sorted, successes)
	sort.Float64s(sorted)
	return AggregateStats{
		AvgMs:       sum(sorted) / float64(len(sorted)),
		MinMs:       sorted[0],
		MaxMs:       sorted[len(sorted)-1],
		P50Ms:       percentile(sorted, 50),
		P95Ms:       percentile(sorted, 95),
		P99Ms:       percentile(sorted, 99),
		LossRate:    lossRate,
		SampleCount: total,
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p / 100.0)
	return sorted[idx]
}

func sum(vals []float64) float64 {
	var s float64
	for _, v := range vals {
		s += v
	}
	return s
}

func jitter(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	avg := sum(vals) / float64(len(vals))
	var variance float64
	for _, v := range vals {
		d := v - avg
		variance += d * d
	}
	// simplified jitter = standard deviation approximation
	return variance / float64(len(vals))
}

// ─── LatencySample JSON serialisation helper ──────────────────────────────

// MarshalJSON marshals the LatencySample.
func (s LatencySample) JSON() string {
	b, _ := json.Marshal(s)
	return string(b)
}

// ─── target helpers ───────────────────────────────────────────────────────

// ParseTarget splits "host:port" → (host, port, error).
func ParseTarget(target string) (host string, port int, err error) {
	h, p, e := net.SplitHostPort(target)
	if e != nil {
		return "", 0, e
	}
	portInt, e := strconv.Atoi(p)
	if e != nil {
		return "", 0, e
	}
	return h, portInt, nil
}
