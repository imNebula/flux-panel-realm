package com.admin.common.task;

import com.admin.common.dto.RealmConfigDto;
import com.admin.common.dto.RealmEndpointDto;
import com.admin.common.dto.WsResult;
import com.admin.common.utils.WebSocketServer;
import com.admin.entity.Forward;
import com.admin.entity.Tunnel;
import com.admin.mapper.ForwardMapper;
import com.admin.mapper.TunnelMapper;
import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import javax.annotation.Resource;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;

@Slf4j
@Component
public class RealmConfigSyncAsync {

    @Resource
    private ForwardMapper forwardMapper;

    @Resource
    private TunnelMapper tunnelMapper;

    /**
     * Rebuild and push the entire Realm configuration for a specific node.
     */
    public void syncNodeConfig(Long nodeId) {
        try {
            RealmConfigDto configDto = new RealmConfigDto();
            List<RealmEndpointDto> endpoints = new ArrayList<>();

            // Find all active forwards where this node is either the inNode or outNode
            QueryWrapper<Forward> forwardQuery = new QueryWrapper<>();
            forwardQuery.eq("status", 1);
            List<Forward> allForwards = forwardMapper.selectList(forwardQuery);

            for (Forward forward : allForwards) {
                Tunnel tunnel = tunnelMapper.selectById(forward.getTunnelId());
                if (tunnel == null || tunnel.getStatus() != 1) {
                    continue;
                }

                // Node is In-Node (Port Forward or Tunnel Forward Entrance)
                if (tunnel.getInNodeId().equals(nodeId)) {
                    RealmEndpointDto endpoint = new RealmEndpointDto();
                    endpoint.setName("forward-" + forward.getId() + "-in");
                    endpoint.setListen(formatListen(listenAddress(tunnel, forward), forward.getInPort()));
                    endpoint.setForward_id(forward.getId());
                    endpoint.setTunnel_id(tunnel.getId());
                    endpoint.setUser_id(forward.getUserId());
                    
                    if (tunnel.getType() == 1) { // Port Forward
                        applyRemoteTargets(endpoint, forward.getRemoteAddr(), forward.getStrategy());
                        endpoint.setListen_transport(tunnel.getProtocol());
                        endpoint.setListen_interface(tunnel.getInterfaceName());
                        endpoint.setInterface_name(forward.getInterfaceName());
                    } else if (tunnel.getType() == 2) { // Tunnel Forward
                        endpoint.setRemote(tunnel.getOutIp() + ":" + forward.getOutPort());
                        endpoint.setRemote_transport(tunnel.getProtocol());
                        endpoint.setInterface_name(firstNonBlank(forward.getInterfaceName(), tunnel.getInterfaceName()));
                    }
                    
                    // UDP support logic
                    if (isUdpEndpoint(tunnel, forward)) {
                        endpoint.setNetwork(udpOnlyNetwork());
                    }
                    
                    endpoints.add(endpoint);
                }

                // Node is Out-Node (Tunnel Forward Exit)
                if (tunnel.getType() == 2 && tunnel.getOutNodeId().equals(nodeId)) {
                    RealmEndpointDto endpoint = new RealmEndpointDto();
                    endpoint.setName("forward-" + forward.getId() + "-out");
                    endpoint.setListen(formatListen(listenAddress(tunnel, forward), forward.getOutPort()));
                    endpoint.setForward_id(forward.getId());
                    endpoint.setTunnel_id(tunnel.getId());
                    endpoint.setUser_id(forward.getUserId());
                    applyRemoteTargets(endpoint, forward.getRemoteAddr(), forward.getStrategy());
                    endpoint.setListen_transport(tunnel.getProtocol());
                    
                    // UDP support logic
                    if (isUdpEndpoint(tunnel, forward)) {
                        endpoint.setNetwork(udpOnlyNetwork());
                    }
                    
                    endpoints.add(endpoint);
                }
            }

            configDto.setEndpoints(endpoints);

            // Push to node
            WsResult result = WebSocketServer.send_msg(nodeId, configDto, "ApplyRealmConfig");
            if (result != null && "OK".equals(result.getMsg())) {
                log.info("Successfully synced Realm config to node {}", nodeId);
            } else {
                log.error("Failed to sync Realm config to node {}: {}", nodeId, result != null ? result.getMsg() : "null");
            }

        } catch (Exception e) {
            log.error("Error syncing Realm config for node {}", nodeId, e);
        }
    }

    private void applyRemoteTargets(RealmEndpointDto endpoint, String rawRemoteAddr, String strategy) {
        List<String> remotes = splitRemoteAddresses(rawRemoteAddr);
        if (remotes.isEmpty()) {
            return;
        }

        endpoint.setRemote(remotes.get(0));
        if (remotes.size() > 1) {
            endpoint.setExtra_remotes(new ArrayList<>(remotes.subList(1, remotes.size())));
            endpoint.setBalance(toRealmBalance(strategy));
        }
    }

    private List<String> splitRemoteAddresses(String rawRemoteAddr) {
        List<String> remotes = new ArrayList<>();
        if (rawRemoteAddr == null) {
            return remotes;
        }
        for (String remote : rawRemoteAddr.split(",")) {
            String trimmed = remote.trim();
            if (!trimmed.isEmpty()) {
                remotes.add(trimmed);
            }
        }
        return remotes;
    }

    private String toRealmBalance(String strategy) {
        if (strategy == null) {
            return null;
        }
        switch (strategy.toLowerCase(Locale.ROOT)) {
            case "round":
            case "roundrobin":
                return "roundrobin";
            case "hash":
            case "iphash":
                return "iphash";
            default:
                return null;
        }
    }

    private boolean isUdpEndpoint(Tunnel tunnel, Forward forward) {
        return "udp".equalsIgnoreCase(tunnel.getProtocol()) || "udp".equalsIgnoreCase(forward.getStrategy());
    }

    private RealmEndpointDto.NetworkConfig udpOnlyNetwork() {
        RealmEndpointDto.NetworkConfig net = new RealmEndpointDto.NetworkConfig();
        net.setUse_udp(true);
        net.setNo_tcp(true);
        return net;
    }

    private String listenAddress(Tunnel tunnel, Forward forward) {
        String addr = isUdpEndpoint(tunnel, forward) ? tunnel.getUdpListenAddr() : tunnel.getTcpListenAddr();
        return firstNonBlank(addr, "0.0.0.0");
    }

    private String formatListen(String host, Integer port) {
        String normalizedHost = firstNonBlank(host, "0.0.0.0");
        if (normalizedHost.contains(":") && !normalizedHost.startsWith("[")) {
            normalizedHost = "[" + normalizedHost + "]";
        }
        return normalizedHost + ":" + port;
    }

    private String firstNonBlank(String first, String fallback) {
        if (first != null && !first.trim().isEmpty()) {
            return first.trim();
        }
        return fallback;
    }
}
