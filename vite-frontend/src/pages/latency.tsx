import { useState, useEffect, useCallback, useRef } from "react";
import { Card, CardBody, CardHeader } from "@heroui/card";
import { Button } from "@heroui/button";
import { Chip } from "@heroui/chip";
import { Spinner } from "@heroui/spinner";
import { Select, SelectItem } from "@heroui/select";
import toast from "react-hot-toast";
import { getLatencySamples, getLatencyAggregates, probeLatency, getNodeList, getForwardList, getTunnelList } from "@/api";

// ─── Types ───────────────────────────────────────────────────────────────────
interface LatencySample {
  id: number; nodeId: number; tunnelId?: number; forwardId?: number;
  protocol: string; probeMode: string; target: string;
  success: number; latencyMs: number; jitterMs?: number; error?: string; sampledAt: number;
}
interface LatencyAggregate {
  id: number; scopeType: string; scopeId: number; window: string;
  avgMs: number; minMs: number; maxMs: number;
  p50Ms: number; p95Ms: number; p99Ms: number;
  lossRate: number; sampleCount: number; createdAt: number;
}
interface ScopeOption { id: number; name: string; type: "node" | "tunnel" | "forward"; }
type TimeRange = "1h" | "6h" | "24h" | "7d";

const TIME_RANGES: { label: string; value: TimeRange; ms: number }[] = [
  { label: "1h", value: "1h", ms: 3600_000 },
  { label: "6h", value: "6h", ms: 6 * 3600_000 },
  { label: "24h", value: "24h", ms: 24 * 3600_000 },
  { label: "7d", value: "7d", ms: 7 * 24 * 3600_000 },
];

// ─── Sparkline ────────────────────────────────────────────────────────────────
function Sparkline({ data, color = "#22d3ee", height = 32 }: { data: number[]; color?: string; height?: number }) {
  if (!data || data.length < 2) return <div style={{ height }} className="flex items-center justify-center text-xs text-default-400">暂无数据</div>;
  const max = Math.max(...data, 1);
  const min = Math.min(...data);
  const range = max - min || 1;
  const w = 120, h = height;
  const pts = data.map((v, i) => `${(i / (data.length - 1)) * w},${h - ((v - min) / range) * (h - 4) - 2}`).join(" ");
  return (
    <svg width={w} height={h} className="overflow-visible">
      <polyline points={pts} fill="none" stroke={color} strokeWidth="1.5" strokeLinejoin="round" strokeLinecap="round" />
      <circle cx={(data.length - 1) / (data.length - 1) * w} cy={h - ((data[data.length - 1] - min) / range) * (h - 4) - 2} r="2.5" fill={color} />
    </svg>
  );
}

// ─── Status badge ─────────────────────────────────────────────────────────────
function StatusBadge({ lossRate, sampleCount }: { lossRate: number; sampleCount: number }) {
  if (sampleCount === 0) return <Chip size="sm" variant="flat" color="default">暂无数据</Chip>;
  if (lossRate >= 1) return <Chip size="sm" variant="flat" color="danger">探测失败</Chip>;
  if (lossRate > 0.5) return <Chip size="sm" variant="flat" color="warning">丢包率高</Chip>;
  if (lossRate > 0.1) return <Chip size="sm" variant="flat" color="warning">部分丢包</Chip>;
  return <Chip size="sm" variant="flat" color="success">正常</Chip>;
}

// ─── Latency card ─────────────────────────────────────────────────────────────
function LatencyCard({ agg, samples, onProbe, probing }: {
  agg: LatencyAggregate | null; samples: LatencySample[]; scopeLabel: string;
  onProbe?: () => void; probing?: boolean;
}) {
  const sparkData = samples.filter(s => s.success === 1).map(s => s.latencyMs).slice(-20);
  const latColor = agg ? (agg.avgMs < 50 ? "#22d3ee" : agg.avgMs < 150 ? "#facc15" : "#f87171") : "#6b7280";
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {agg ? <StatusBadge lossRate={agg.lossRate ?? 1} sampleCount={agg.sampleCount} /> : <Chip size="sm" variant="flat" color="default">暂无数据</Chip>}
          {onProbe && (
            <Button size="sm" variant="flat" isDisabled={probing} isLoading={probing} onPress={onProbe} className="h-6 text-xs px-2">探测</Button>
          )}
        </div>
        {agg && <span className="text-xs text-default-400">{agg.sampleCount} 次采样</span>}
      </div>
      <div className="flex items-end gap-4">
        <div>
          <div className="text-2xl font-bold tabular-nums" style={{ color: latColor }}>
            {agg ? `${agg.avgMs?.toFixed(1)}ms` : "--"}
          </div>
          <div className="text-xs text-default-500">平均延迟</div>
        </div>
        <div className="flex-1">
          <Sparkline data={sparkData} color={latColor} />
        </div>
      </div>
      {agg && (
        <div className="grid grid-cols-5 gap-1 text-center">
          {[["min", agg.minMs], ["p50", agg.p50Ms], ["p95", agg.p95Ms], ["p99", agg.p99Ms], ["max", agg.maxMs]].map(([label, val]) => (
            <div key={label as string} className="bg-default-100 rounded-lg p-1.5">
              <div className="text-xs font-mono font-semibold">{(val as number)?.toFixed(0) ?? "--"}</div>
              <div className="text-[10px] text-default-500">{label}</div>
            </div>
          ))}
        </div>
      )}
      {agg && (
        <div className="flex gap-3 text-xs text-default-500">
          <span>丢包率: <span className={agg.lossRate > 0.1 ? "text-warning" : "text-success"}>{((agg.lossRate ?? 0) * 100).toFixed(1)}%</span></span>
          <span>抖动: <span>{agg.minMs && agg.maxMs ? `${(agg.maxMs - agg.minMs).toFixed(0)}ms` : "--"}</span></span>
        </div>
      )}
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────
export default function LatencyPage() {
  const [timeRange, setTimeRange] = useState<TimeRange>("1h");
  const [scopeType, setScopeType] = useState<"node" | "tunnel" | "forward">("node");
  const [scopeId, setScopeId] = useState<number | null>(null);
  const [scopeOptions, setScopeOptions] = useState<ScopeOption[]>([]);
  const [samples, setSamples] = useState<LatencySample[]>([]);
  const [aggregates, setAggregates] = useState<LatencyAggregate[]>([]);
  const [loading, setLoading] = useState(false);
  const [probing, setProbing] = useState<Record<number, boolean>>({});
  const intervalRef = useRef<any>(null);

  // Load scope options
  useEffect(() => {
    const load = async () => {
      try {
        if (scopeType === "node") {
          const res = await getNodeList();
          if (res.code === 0) setScopeOptions(res.data.map((n: any) => ({ id: n.id, name: n.name, type: "node" })));
        } else if (scopeType === "tunnel") {
          const res = await getTunnelList();
          if (res.code === 0) setScopeOptions(res.data.map((t: any) => ({ id: t.id, name: t.name, type: "tunnel" })));
        } else {
          const res = await getForwardList();
          if (res.code === 0) setScopeOptions(res.data.map((f: any) => ({ id: f.id, name: f.name || `端口${f.inPort}`, type: "forward" })));
        }
        setScopeId(null);
      } catch {}
    };
    load();
  }, [scopeType]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    const rangeMs = TIME_RANGES.find(r => r.value === timeRange)?.ms ?? 3600_000;
    const sinceMs = Date.now() - rangeMs;
    const params: any = { limit: 200 };
    if (scopeId !== null) params[`${scopeType}Id`] = scopeId;
    try {
      const [sRes, aRes] = await Promise.all([
        getLatencySamples({ ...params, sinceMs }),
        getLatencyAggregates({ scopeType: scopeId ? scopeType : undefined, scopeId: scopeId ?? undefined, window: timeRange === "1h" ? "minute" : timeRange === "7d" ? "day" : "hour" }),
      ]);
      if (sRes.code === 0) setSamples(sRes.data || []);
      if (aRes.code === 0) setAggregates(aRes.data || []);
    } catch {
      toast.error("获取延迟数据失败");
    } finally {
      setLoading(false);
    }
  }, [timeRange, scopeType, scopeId]);

  useEffect(() => {
    fetchData();
    intervalRef.current = setInterval(fetchData, 30_000);
    return () => clearInterval(intervalRef.current);
  }, [fetchData]);

  const handleProbe = async (nodeId: number, target: string, tunnelId?: number, forwardId?: number) => {
    setProbing(p => ({ ...p, [nodeId]: true }));
    try {
      const res = await probeLatency({ nodeId, target, count: 3, timeout: 3000, tunnelId, forwardId });
      if (res.code === 0) {
        toast.success(`探测完成: ${res.data?.latencyMs?.toFixed?.(1) ?? "--"}ms`);
        fetchData();
      } else {
        toast.error(res.msg || "探测失败");
      }
    } catch {
      toast.error("探测请求失败");
    } finally {
      setProbing(p => ({ ...p, [nodeId]: false }));
    }
  };

  // Build cards: one per (scopeId) in aggregates or nodes
  const cards = (() => {
    const window = timeRange === "1h" ? "minute" : timeRange === "7d" ? "day" : "hour";
    const filteredAggs = aggregates.filter(a => a.window === window && a.scopeType === scopeType);
    if (filteredAggs.length === 0 && samples.length === 0) return [];
    const scopeIds = new Set([...filteredAggs.map(a => a.scopeId), ...samples.map(s => s[`${scopeType}Id` as keyof LatencySample] as number).filter(Boolean)]);
    return Array.from(scopeIds).map(id => {
      const agg = filteredAggs.find(a => a.scopeId === id) ?? null;
      const scopeSamples = samples.filter(s => (s as any)[`${scopeType}Id`] === id);
      const label = scopeOptions.find(o => o.id === id)?.name ?? `${scopeType} #${id}`;
      return { id, label, agg, samples: scopeSamples };
    });
  })();

  return (
    <div className="px-3 lg:px-6 py-8">
      {/* Header */}
      <div className="flex flex-wrap items-center gap-3 mb-6">
        <h1 className="text-xl font-bold flex-1">延迟监控</h1>
        {/* Time Range */}
        <div className="flex items-center gap-1 bg-default-100 rounded-xl p-1">
          {TIME_RANGES.map(r => (
            <button key={r.value} onClick={() => setTimeRange(r.value)}
              className={`px-3 py-1 rounded-lg text-sm font-medium transition-all ${timeRange === r.value ? "bg-white dark:bg-default-800 shadow text-primary" : "text-default-500 hover:text-foreground"}`}>
              {r.label}
            </button>
          ))}
        </div>
        {/* Scope Type */}
        <div className="flex items-center gap-1 bg-default-100 rounded-xl p-1">
          {(["node", "tunnel", "forward"] as const).map(t => (
            <button key={t} onClick={() => setScopeType(t)}
              className={`px-3 py-1 rounded-lg text-sm font-medium transition-all ${scopeType === t ? "bg-white dark:bg-default-800 shadow text-primary" : "text-default-500 hover:text-foreground"}`}>
              {t === "node" ? "节点" : t === "tunnel" ? "隧道" : "转发"}
            </button>
          ))}
        </div>
        {/* Scope Filter */}
        {scopeOptions.length > 0 && (
          <Select size="sm" placeholder="全部" className="w-40" selectedKeys={scopeId ? [String(scopeId)] : []}
            onSelectionChange={keys => { const v = Array.from(keys)[0]; setScopeId(v ? Number(v) : null); }}>
            {scopeOptions.map(o => <SelectItem key={String(o.id)}>{o.name}</SelectItem>)}
          </Select>
        )}
        <Button size="sm" variant="flat" onPress={fetchData} isLoading={loading}>刷新</Button>
      </div>

      {/* Empty state */}
      {loading && cards.length === 0 && (
        <div className="flex justify-center py-20"><Spinner size="lg" /></div>
      )}
      {!loading && cards.length === 0 && (
        <Card className="shadow-sm border border-divider">
          <CardBody className="text-center py-16">
            <div className="flex flex-col items-center gap-4">
              <div className="w-16 h-16 bg-default-100 rounded-full flex items-center justify-center text-3xl">📡</div>
              <div>
                <h3 className="text-lg font-semibold">暂无延迟数据</h3>
                <p className="text-default-500 text-sm mt-1">点击节点卡片上的「探测」按钮，或等待自动采样数据积累</p>
              </div>
              <div className="flex gap-2 flex-wrap justify-center text-xs text-default-400">
                <span className="bg-default-100 px-2 py-1 rounded">采集方式: TCP Connect</span>
                <span className="bg-default-100 px-2 py-1 rounded">更新间隔: 30秒</span>
                <span className="bg-default-100 px-2 py-1 rounded">聚合窗口: 分钟/小时/天</span>
              </div>
            </div>
          </CardBody>
        </Card>
      )}

      {/* Cards grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {cards.map(card => (
          <Card key={card.id} className="shadow-sm border border-divider hover:shadow-md transition-shadow duration-200">
            <CardHeader className="pb-1">
              <div className="flex justify-between items-start w-full">
                <div>
                  <h3 className="font-semibold text-sm truncate max-w-44" title={card.label}>{card.label}</h3>
                  <span className="text-xs text-default-400">{scopeType === "node" ? "节点" : scopeType === "tunnel" ? "隧道" : "转发"} #{card.id}</span>
                </div>
                <Chip size="sm" variant="flat" color="primary" className="text-xs">{timeRange}</Chip>
              </div>
            </CardHeader>
            <CardBody className="pt-0">
              <LatencyCard
                agg={card.agg}
                samples={card.samples}
                scopeLabel={card.label}
                probing={probing[card.id]}
                onProbe={scopeType === "node" ? () => {
                  const s = card.samples[0];
                  if (s?.target) handleProbe(card.id, s.target);
                  else toast.error("无探测目标，请先在转发/隧道设置中配置");
                } : undefined}
              />
            </CardBody>
          </Card>
        ))}
      </div>

      {/* Status legend */}
      <div className="mt-6 flex flex-wrap gap-3 text-xs text-default-400">
        <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-success inline-block" />正常 (&lt;10% 丢包)</span>
        <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-warning inline-block" />部分丢包</span>
        <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-danger inline-block" />探测失败</span>
        <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-default-300 inline-block" />暂无数据</span>
        <span className="ml-auto">延迟颜色: <span className="text-cyan-400">优秀(&lt;50ms)</span> <span className="text-yellow-400">良好(&lt;150ms)</span> <span className="text-red-400">较高</span></span>
      </div>
    </div>
  );
}
