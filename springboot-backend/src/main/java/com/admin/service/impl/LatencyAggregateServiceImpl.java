package com.admin.service.impl;

import com.admin.entity.LatencyAggregate;
import com.admin.mapper.LatencyAggregateMapper;
import com.admin.service.LatencyAggregateService;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import org.springframework.stereotype.Service;

@Service
public class LatencyAggregateServiceImpl extends ServiceImpl<LatencyAggregateMapper, LatencyAggregate> implements LatencyAggregateService {
}
