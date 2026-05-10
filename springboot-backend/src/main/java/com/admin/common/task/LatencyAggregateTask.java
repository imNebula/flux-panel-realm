package com.admin.common.task;

import com.admin.entity.LatencyAggregate;
import com.admin.entity.LatencySample;
import com.admin.service.LatencyAggregateService;
import com.admin.service.LatencySampleService;
import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import lombok.extern.slf4j.Slf4j;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import javax.annotation.Resource;
import java.math.BigDecimal;
import java.math.RoundingMode;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Aggregates latency_sample rows into latency_aggregate records for
 * minute/hour/day windows.
 *
 * Computes: avg_ms, min_ms, max_ms, p50_ms, p95_ms, p99_ms, loss_rate, sample_count.
 * Respects retention_days to clean up old samples and aggregates.
 *
 * Scheduled:
 *   - Every 5 minutes: aggregate last 10 minutes of samples into the "minute" window.
 *   - Every hour: aggregate the last 2 hours into the "hour" window.
 *   - Every day at 01:00: aggregate into the "day" window and purge old data.
 */
@Component
@Slf4j
public class LatencyAggregateTask {

    private static final int DEFAULT_RETENTION_DAYS = 7;

    @Resource
    private LatencySampleService latencySampleService;

    @Resource
    private LatencyAggregateService latencyAggregateService;

    /** Aggregate minute window – run every 5 minutes. */
    @Scheduled(fixedDelay = 300_000, initialDelay = 60_000)
    public void aggregateMinute() {
        long cutoff = System.currentTimeMillis() - 10 * 60 * 1000L; // last 10 min
        aggregateWindow("minute", cutoff, System.currentTimeMillis());
    }

    /** Aggregate hour window – run every hour. */
    @Scheduled(fixedDelay = 3_600_000, initialDelay = 120_000)
    public void aggregateHour() {
        long cutoff = System.currentTimeMillis() - 2 * 3600 * 1000L;
        aggregateWindow("hour", cutoff, System.currentTimeMillis());
    }

    /** Aggregate day window and purge old data – run once a day at startup + 24h. */
    @Scheduled(fixedDelay = 86_400_000, initialDelay = 300_000)
    public void aggregateDayAndPurge() {
        long cutoff = System.currentTimeMillis() - 25 * 3600 * 1000L;
        aggregateWindow("day", cutoff, System.currentTimeMillis());
        purgeOldData();
    }

    /**
     * Aggregates samples within [sinceMs, nowMs] grouped by (scope_type, scope_id, window).
     * Scopes: node, tunnel, forward.
     */
    private void aggregateWindow(String window, long sinceMs, long nowMs) {
        try {
            QueryWrapper<LatencySample> qw = new QueryWrapper<>();
            qw.ge("sampled_at", sinceMs).le("sampled_at", nowMs);
            List<LatencySample> samples = latencySampleService.list(qw);
            if (samples.isEmpty()) return;

            // Group by (scopeType, scopeId)
            Map<String, List<LatencySample>> byScope = new LinkedHashMap<>();
            for (LatencySample s : samples) {
                // node scope
                addToGroup(byScope, "node", s.getNodeId(), s);
                // tunnel scope
                if (s.getTunnelId() != null) {
                    addToGroup(byScope, "tunnel", s.getTunnelId(), s);
                }
                // forward scope
                if (s.getForwardId() != null) {
                    addToGroup(byScope, "forward", s.getForwardId(), s);
                }
            }

            for (Map.Entry<String, List<LatencySample>> entry : byScope.entrySet()) {
                String[] parts = entry.getKey().split(":");
                String scopeType = parts[0];
                Long scopeId = Long.valueOf(parts[1]);
                List<LatencySample> group = entry.getValue();
                LatencyAggregate agg = computeAggregate(scopeType, scopeId, window, group, nowMs);
                if (agg != null) {
                    latencyAggregateService.save(agg);
                }
            }
            log.debug("延迟聚合完成 window={} samples={}", window, samples.size());
        } catch (Exception e) {
            log.error("延迟聚合失败 window={}: {}", window, e.getMessage());
        }
    }

    private void addToGroup(Map<String, List<LatencySample>> map, String scopeType, Long scopeId, LatencySample s) {
        if (scopeId == null) return;
        String key = scopeType + ":" + scopeId;
        map.computeIfAbsent(key, k -> new ArrayList<>()).add(s);
    }

    private LatencyAggregate computeAggregate(String scopeType, Long scopeId, String window,
                                               List<LatencySample> samples, long nowMs) {
        if (samples.isEmpty()) return null;
        int total = samples.size();
        List<Double> successes = samples.stream()
                .filter(s -> s.getSuccess() != null && s.getSuccess() == 1 && s.getLatencyMs() != null)
                .map(s -> s.getLatencyMs().doubleValue())
                .sorted()
                .collect(Collectors.toList());

        LatencyAggregate agg = new LatencyAggregate();
        agg.setScopeType(scopeType);
        agg.setScopeId(scopeId);
        agg.setWindow(window);
        agg.setSampleCount(total);
        double lossRate = total > 0 ? 1.0 - (double) successes.size() / total : 1.0;
        agg.setLossRate(BigDecimal.valueOf(lossRate).setScale(4, RoundingMode.HALF_UP));
        agg.setCreatedAt(nowMs);

        if (!successes.isEmpty()) {
            double avg = successes.stream().mapToDouble(Double::doubleValue).average().orElse(0);
            agg.setAvgMs(bd(avg));
            agg.setMinMs(bd(successes.get(0)));
            agg.setMaxMs(bd(successes.get(successes.size() - 1)));
            agg.setP50Ms(bd(percentile(successes, 50)));
            agg.setP95Ms(bd(percentile(successes, 95)));
            agg.setP99Ms(bd(percentile(successes, 99)));
        }
        return agg;
    }

    private BigDecimal bd(double v) {
        return BigDecimal.valueOf(v).setScale(2, RoundingMode.HALF_UP);
    }

    private double percentile(List<Double> sorted, double p) {
        if (sorted.isEmpty()) return 0;
        int idx = (int) Math.ceil(sorted.size() * p / 100.0) - 1;
        return sorted.get(Math.max(0, Math.min(idx, sorted.size() - 1)));
    }

    /** Deletes samples and aggregates older than retention_days. */
    private void purgeOldData() {
        try {
            long cutoff = System.currentTimeMillis() - (long) DEFAULT_RETENTION_DAYS * 86_400_000L;
            QueryWrapper<LatencySample> sampleQw = new QueryWrapper<>();
            sampleQw.lt("sampled_at", cutoff);
            int removedSamples = latencySampleService.count(sampleQw);
            latencySampleService.remove(sampleQw);

            QueryWrapper<LatencyAggregate> aggQw = new QueryWrapper<>();
            aggQw.lt("created_at", cutoff);
            int removedAggs = latencyAggregateService.count(aggQw);
            latencyAggregateService.remove(aggQw);

            log.info("延迟数据清理: 删除 {} 条采样, {} 条聚合 (保留 {} 天)", removedSamples, removedAggs, DEFAULT_RETENTION_DAYS);
        } catch (Exception e) {
            log.error("延迟数据清理失败: {}", e.getMessage());
        }
    }
}
