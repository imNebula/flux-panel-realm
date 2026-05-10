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
@EqualsAndHashCode(callSuper = true)
public class Node extends BaseEntity {

    private static final long serialVersionUID = 1L;

    private String name;

    private String secret;

    private String ip;

    private String serverIp;

    private String version;

    private Integer portSta;

    private Integer portEnd;

    private Integer http;

    private Integer tls;

    private Integer socks;

    private String agentVersion;

    private String realmVersion;

    private String realmBinaryPath;

    private String realmConfigDir;

    private String realmProcessName;

    private String realmServiceName;

    private String agentProcessName;

    private String instanceName;

    private String os;

    private String distro;

    private String osVersion;

    private String arch;

    private String libc;

    private String initSystem;

    private String containerType;

    private String capabilitiesJson;

    private String runningProcessesJson;

    private String configHash;

    private Integer endpointCount;

    private Integer activeForwardCount;

    private Integer activeTunnelCount;

    private String lastApplyId;

    private Integer lastApplyStatus;

    private String lastApplyError;

    private String lastApplyJson;

}
