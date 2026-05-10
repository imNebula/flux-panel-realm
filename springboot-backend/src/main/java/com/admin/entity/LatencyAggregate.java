package com.admin.entity;

import lombok.Data;
import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;

import java.math.BigDecimal;

@Data
public class LatencyAggregate {
    @TableId(value = "id", type = IdType.AUTO)
    private Long id;
    private String scopeType;
    private Long scopeId;
    private String window;
    private BigDecimal avgMs;
    private BigDecimal minMs;
    private BigDecimal maxMs;
    private BigDecimal p50Ms;
    private BigDecimal p95Ms;
    private BigDecimal p99Ms;
    private BigDecimal lossRate;
    private Integer sampleCount;
    private Long createdAt;
}
