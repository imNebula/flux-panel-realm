package com.admin.common.dto;

import lombok.Data;

/**
 * WebSocket command result returned by {@code WebSocketServer.send_msg}.
 * "OK" in {@code msg} indicates success; any other value is an error.
 */
@Data
public class WsResult {
    private Integer code;
    private String msg;
    private Object data;
}
