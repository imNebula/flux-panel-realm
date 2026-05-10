package com.admin.controller;

import com.admin.common.annotation.RequireRole;
import com.admin.common.aop.LogAnnotation;
import com.admin.common.dto.GostDto;
import com.admin.common.lang.R;
import com.admin.common.utils.WebSocketServer;
import com.admin.entity.LatencySample;
import com.admin.service.LatencyAggregateService;
import com.admin.service.LatencySampleService;
import com.alibaba.fastjson.JSONObject;
import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import org.springframework.web.bind.annotation.*;

import javax.annotation.Resource;
import java.math.BigDecimal;
import java.util.Map;

@RestController
@CrossOrigin
@RequestMapping("/api/v1/latency")
public class LatencyController extends BaseController {

    @Resource
    LatencySampleService latencySampleService;

    @Resource
    LatencyAggregateService latencyAggregateService;

    @RequireRole
    @GetMapping("/samples")
    public R samples(@RequestParam(required = false) Long nodeId,
                     @RequestParam(required = false) Long tunnelId,
                     @RequestParam(required = false) Long forwardId,
                     @RequestParam(required = false) Long sinceMs,
                     @RequestParam(defaultValue = "100") Integer limit) {
        QueryWrapper<LatencySample> query = new QueryWrapper<>();
        if (nodeId != null)    query.eq("node_id", nodeId);
        if (tunnelId != null)  query.eq("tunnel_id", tunnelId);
        if (forwardId != null) query.eq("forward_id", forwardId);
        if (sinceMs != null)   query.ge("sampled_at", sinceMs);
        query.orderByDesc("sampled_at").last("limit " + Math.min(Math.max(limit, 1), 500));
        return R.ok(latencySampleService.list(query));
    }

    @RequireRole
    @GetMapping("/aggregates")
    public R aggregates(@RequestParam(required = false) String scopeType,
                        @RequestParam(required = false) Long scopeId,
                        @RequestParam(defaultValue = "hour") String window) {
        QueryWrapper<?> query = new QueryWrapper<>();
        if (scopeType != null) query.eq("scope_type", scopeType);
        if (scopeId != null) query.eq("scope_id", scopeId);
        query.eq("window", window).orderByDesc("created_at").last("limit 200");
        return R.ok(latencyAggregateService.list((QueryWrapper) query));
    }

    @LogAnnotation
    @RequireRole
    @PostMapping("/probe")
    public R probe(@RequestBody Map<String, Object> params) {
        Long nodeId = Long.valueOf(params.get("nodeId").toString());
        String target = params.get("target").toString();
        String[] parts = target.split(":");
        if (parts.length < 2) return R.err("target 必须是 host:port");

        JSONObject tcpPingData = new JSONObject();
        tcpPingData.put("ip", parts[0]);
        tcpPingData.put("port", Integer.parseInt(parts[parts.length - 1]));
        tcpPingData.put("count", params.getOrDefault("count", 2));
        tcpPingData.put("timeout", params.getOrDefault("timeout", 3000));

        GostDto result = WebSocketServer.send_msg(nodeId, tcpPingData, "TcpPing");
        if (!"OK".equals(result.getMsg())) {
            return R.err(result.getMsg());
        }
        JSONObject data = (JSONObject) result.getData();
        LatencySample sample = new LatencySample();
        sample.setNodeId(nodeId);
        sample.setTunnelId(params.get("tunnelId") == null ? null : Long.valueOf(params.get("tunnelId").toString()));
        sample.setForwardId(params.get("forwardId") == null ? null : Long.valueOf(params.get("forwardId").toString()));
        sample.setProtocol("tcp");
        sample.setProbeMode("tcp_connect");
        sample.setTarget(target);
        sample.setSuccess(data.getBooleanValue("success") ? 1 : 0);
        sample.setLatencyMs(BigDecimal.valueOf(data.getDoubleValue("averageTime")));
        sample.setError(data.getString("errorMessage"));
        sample.setSampledAt(System.currentTimeMillis());
        latencySampleService.save(sample);
        return R.ok(sample);
    }

    @LogAnnotation
    @RequireRole
    @PatchMapping("/config")
    public R config(@RequestBody Map<String, Object> params) {
        return R.ok("latency config accepted; per-forward/per-tunnel persistence is handled by their update APIs");
    }
}
