package com.admin.entity;

import lombok.Data;
import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;

import java.math.BigDecimal;

@Data
public class LatencySample {
    @TableId(value = "id", type = IdType.AUTO)
    private Long id;
    private Long nodeId;
    private Long tunnelId;
    private Long forwardId;
    private String protocol;
    private String probeMode;
    private String target;
    private Integer success;
    private BigDecimal latencyMs;
    private BigDecimal jitterMs;
    private String error;
    private Long sampledAt;
}
