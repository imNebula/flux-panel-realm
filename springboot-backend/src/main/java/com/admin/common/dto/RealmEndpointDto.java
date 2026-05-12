package com.admin.common.dto;

import lombok.Data;
import java.util.List;

@Data
public class RealmEndpointDto {
    private String name;
    private Long forward_id;
    private Long tunnel_id;
    private Integer user_id;
    private String listen;
    private String remote;
    private List<String> extra_remotes;
    private String balance;
    private String through;
    private String interface_name;
    private String listen_interface;
    private String listen_transport;
    private String remote_transport;
    private NetworkConfig network;

    @Data
    public static class NetworkConfig {
        private Boolean use_udp;
        private Boolean no_tcp;
        private Boolean send_proxy;
        private Boolean accept_proxy;
    }
}
