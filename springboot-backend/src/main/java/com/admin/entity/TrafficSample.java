package com.admin.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

/**
 * Per-endpoint traffic sample collected from nftables/iptables/procfs on the agent.
 * Maps to the traffic_sample table created in V2__realm_migration.sql.
 */
@Data
@TableName("traffic_sample")
public class TrafficSample {
    @TableId(value = "id", type = IdType.AUTO)
    private Long id;
    private Long nodeId;
    private Long tunnelId;
    private Long forwardId;
    private Long userId;
    private String listenAddr;
    private Integer listenPort;
    /** tcp / udp */
    private String protocol;
    private Long inBytes;
    private Long outBytes;
    private Long totalBytes;
    /** billing_bytes = totalBytes × traffic_ratio from tunnel config */
    private Long billingBytes;
    /** epoch milliseconds */
    private Long sampleTime;
    /** collection method: nftables / iptables / procfs / none */
    private String method;
}
