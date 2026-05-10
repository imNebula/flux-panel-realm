package com.admin.entity;

import lombok.Data;
import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;

@Data
public class ApplyHistory {
    @TableId(value = "id", type = IdType.AUTO)
    private Long id;
    private Long nodeId;
    private String applyId;
    private String changedResources;
    private String configHashBefore;
    private String configHashAfter;
    private String validationResult;
    private String action;
    private Long durationMs;
    private Integer success;
    private String errorMessage;
}
