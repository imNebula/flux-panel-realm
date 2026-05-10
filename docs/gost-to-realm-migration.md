# Gost to Realm migration

This project now treats Realm v2 as the forwarding engine. The old Gost WebSocket commands are accepted by `flux-realm-agent` as a compatibility layer and translated to Realm `endpoints`.

## Realm configuration shape

The generated JSON uses Realm v2 fields:

- `log`
- `dns`
- `network`
- `endpoints`
- `endpoint.listen`
- `endpoint.remote`
- `endpoint.extra_remotes`
- `endpoint.balance`
- `endpoint.through`
- `endpoint.interface`
- `endpoint.listen_interface`
- `endpoint.listen_transport`
- `endpoint.remote_transport`
- `endpoint.network`

Configs are written with: temporary file, `fsync`, `rename`, directory `fsync`, parse/validate, then process apply.

## Compatibility matrix

| Legacy Gost concept | Realm mapping | Status |
| --- | --- | --- |
| TCP local forward | `endpoint.listen` + `endpoint.remote` | Supported |
| UDP local forward | endpoint with `network.no_tcp=true`, `network.use_udp=true` | Supported |
| TCP+UDP same port | two compatibility services collapse to same listen port/protocol-specific network | Supported |
| Multiple remote nodes | `remote` + `extra_remotes` | Supported |
| Gost `round/roundrobin` selector | `balance=roundrobin` | Supported |
| Gost `hash/iphash` selector | `balance=iphash` | Supported |
| Gost `interface` metadata | `interface` or `listen_interface` | Supported |
| Gost `tls/ws/wss` dialer/listener | `remote_transport` / `listen_transport` | Supported when Realm build has transport feature |
| Gost relay tunnel chain | ingress endpoint remote points to remote Realm relay endpoint | Supported with transport downgrade |
| PROXY protocol | Realm `network.send_proxy/accept_proxy` | Supported by new Realm config path |
| MPTCP | Realm `send_mptcp/accept_mptcp` | Capability gated |
| Gost HTTP/SOCKS proxy modes | none | Unsupported, retained as migrated legacy record |
| Gost QUIC/KCP/SSH/SS/HTTP2/HTTP3/GRPC/TUN/TAP | none | Unsupported, retained and marked unsupported |
| Gost limiter objects | no native Realm equivalent | Accepted as compatibility no-op; quota enforcement remains backend-side |
| Gost per-service traffic API | Linux nftables/iptables/procfs counters | Supported with degraded modes |

Unsupported migrated rows must remain in the database and be displayed as requiring user action. The backend and frontend must not offer unsupported protocols when the node capability matrix says they are unavailable.

## Apply strategy

Realm does not expose a hot-reload API compatible with this panel. `flux-realm-agent` implements equivalent realtime apply:

- single process: generate config, validate, atomic replace, graceful restart
- per-tunnel/per-forward: reserved process modes with isolated config/pid/service names
- failed apply: restore previous config and restart previous process

Process stop is always based on pidfile/service/instance metadata. Broad commands such as `pkill realm` are forbidden.

## Traffic and latency

Realm has no built-in per-endpoint stats API. The agent reports `traffic_stats_method` as `nftables`, `iptables`, `procfs`, or `none`. In containers/LXC without required capabilities, the node is marked limited/unsupported and the frontend must show the reason.

Latency probes use TCP connect by default. UDP probes require an explicit echo/probe target.
