package com.admin.common.dto;

import lombok.Data;
import java.util.List;

@Data
public class RealmConfigDto {
    private List<RealmEndpointDto> endpoints;
}
