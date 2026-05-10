package com.admin.controller;

import com.admin.common.annotation.RequireRole;
import com.admin.common.lang.R;
import com.admin.entity.TrafficSample;
import com.admin.service.TrafficSampleService;
import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import org.springframework.web.bind.annotation.*;

import javax.annotation.Resource;
import java.util.List;

/**
 * Per-endpoint traffic statistics API.
 * Provides access to traffic samples collected from nftables/iptables/procfs on agents.
 */
@RestController
@CrossOrigin
@RequestMapping("/api/v1/traffic")
public class TrafficController extends BaseController {

    @Resource
    TrafficSampleService trafficSampleService;

    /**
     * List traffic samples with optional filtering.
     * GET /api/v1/traffic/samples?nodeId=&forwardId=&tunnelId=&userId=&limit=100
     */
    @RequireRole
    @GetMapping("/samples")
    public R samples(
            @RequestParam(required = false) Long nodeId,
            @RequestParam(required = false) Long forwardId,
            @RequestParam(required = false) Long tunnelId,
            @RequestParam(required = false) Long userId,
            @RequestParam(required = false) String protocol,
            @RequestParam(required = false) Long sinceMs,
            @RequestParam(defaultValue = "100") Integer limit) {
        QueryWrapper<TrafficSample> qw = new QueryWrapper<>();
        if (nodeId != null)    qw.eq("node_id", nodeId);
        if (forwardId != null) qw.eq("forward_id", forwardId);
        if (tunnelId != null)  qw.eq("tunnel_id", tunnelId);
        if (userId != null)    qw.eq("user_id", userId);
        if (protocol != null)  qw.eq("protocol", protocol);
        if (sinceMs != null)   qw.ge("sample_time", sinceMs);
        qw.orderByDesc("sample_time").last("LIMIT " + Math.min(Math.max(limit, 1), 1000));
        List<TrafficSample> list = trafficSampleService.list(qw);
        return R.ok(list);
    }

    /**
     * Summary: aggregate total bytes per forward for a time range.
     * GET /api/v1/traffic/summary?forwardId=&sinceMs=
     */
    @RequireRole
    @GetMapping("/summary")
    public R summary(
            @RequestParam(required = false) Long forwardId,
            @RequestParam(required = false) Long nodeId,
            @RequestParam(required = false) Long sinceMs) {
        QueryWrapper<TrafficSample> qw = new QueryWrapper<>();
        if (forwardId != null) qw.eq("forward_id", forwardId);
        if (nodeId != null)    qw.eq("node_id", nodeId);
        if (sinceMs != null)   qw.ge("sample_time", sinceMs);
        qw.select("node_id", "forward_id", "tunnel_id", "protocol",
                  "SUM(in_bytes) AS in_bytes", "SUM(out_bytes) AS out_bytes",
                  "SUM(total_bytes) AS total_bytes", "SUM(billing_bytes) AS billing_bytes",
                  "COUNT(*) AS sample_count", "MIN(sample_time) AS first_sample",
                  "MAX(sample_time) AS last_sample");
        if (forwardId != null) {
            qw.groupBy("forward_id", "protocol");
        } else {
            qw.groupBy("node_id", "protocol");
        }
        // MyBatis-Plus does not support arbitrary projection with maps easily.
        // Return raw samples and let the frontend aggregate for now.
        // TODO: implement a custom mapper query for group-by aggregate.
        List<TrafficSample> list = trafficSampleService.list(new QueryWrapper<TrafficSample>()
                .eq(forwardId != null, "forward_id", forwardId)
                .eq(nodeId != null, "node_id", nodeId)
                .ge(sinceMs != null, "sample_time", sinceMs)
                .orderByDesc("sample_time").last("LIMIT 500"));
        // Compute summary in Java.
        long inBytes = 0, outBytes = 0, totalBytes = 0, billingBytes = 0;
        for (TrafficSample s : list) {
            inBytes     += s.getInBytes()     != null ? s.getInBytes()     : 0;
            outBytes    += s.getOutBytes()    != null ? s.getOutBytes()    : 0;
            totalBytes  += s.getTotalBytes()  != null ? s.getTotalBytes()  : 0;
            billingBytes+= s.getBillingBytes() != null ? s.getBillingBytes(): 0;
        }
        return R.ok(java.util.Map.of(
                "in_bytes", inBytes,
                "out_bytes", outBytes,
                "total_bytes", totalBytes,
                "billing_bytes", billingBytes,
                "sample_count", list.size()
        ));
    }
}
