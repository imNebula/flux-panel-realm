package com.admin.entity;

import java.io.Serializable;
import lombok.Data;
import lombok.EqualsAndHashCode;

/**
 * <p>
 * 
 * </p>
 *
 * @author QAQ
 * @since 2025-06-03
 */
@Data
@EqualsAndHashCode(callSuper = false)
public class Forward extends BaseEntity{

    private static final long serialVersionUID = 1L;

    private Integer userId;

    private String userName;

    private String name;

    private Integer tunnelId;

    private Integer inPort;

    private Integer outPort;

    private String remoteAddr;

    private String interfaceName;

    private String strategy;

    private Long inFlow;

    private Long outFlow;

    private Integer inx;

    private String listenTransport;

    private String remoteTransport;

    private String proxyProtocol;

    private String balanceStrategy;

    private String extraRemotes;

    private String throughAddr;

    private String processMode;

    private String processName;

    private String statsMethod;

    private Integer latencySampleEnabled;

    private Integer latencySampleInterval;

}
