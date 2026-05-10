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
                    endpoint.setListen("0.0.0.0:" + forward.getInPort());
                    
                    if (tunnel.getType() == 1) { // Port Forward
                        endpoint.setRemote(forward.getRemoteAddr());
                        endpoint.setListen_transport(tunnel.getProtocol());
                        endpoint.setListen_interface(tunnel.getInterfaceName());
                    } else if (tunnel.getType() == 2) { // Tunnel Forward
                        endpoint.setRemote(tunnel.getOutIp() + ":" + forward.getOutPort());
                        endpoint.setRemote_transport(tunnel.getProtocol());
                        endpoint.setInterface_name(tunnel.getInterfaceName());
                    }
                    
                    // UDP support logic
                    if ("udp".equalsIgnoreCase(tunnel.getProtocol()) || "udp".equalsIgnoreCase(forward.getStrategy())) {
                        RealmEndpointDto.NetworkConfig net = new RealmEndpointDto.NetworkConfig();
                        net.setUse_udp(true);
                        net.setNo_tcp(true);
                        endpoint.setNetwork(net);
                    }
                    
                    endpoints.add(endpoint);
                }

                // Node is Out-Node (Tunnel Forward Exit)
                if (tunnel.getType() == 2 && tunnel.getOutNodeId().equals(nodeId)) {
                    RealmEndpointDto endpoint = new RealmEndpointDto();
                    endpoint.setListen("0.0.0.0:" + forward.getOutPort());
                    endpoint.setRemote(forward.getRemoteAddr());
                    endpoint.setListen_transport(tunnel.getProtocol());
                    
                    // UDP support logic
                    if ("udp".equalsIgnoreCase(forward.getStrategy())) {
                        RealmEndpointDto.NetworkConfig net = new RealmEndpointDto.NetworkConfig();
                        net.setUse_udp(true);
                        net.setNo_tcp(true);
                        endpoint.setNetwork(net);
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
}
