package com.admin.service.impl;

import com.admin.entity.TrafficSample;
import com.admin.mapper.TrafficSampleMapper;
import com.admin.service.TrafficSampleService;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import org.springframework.stereotype.Service;

@Service
public class TrafficSampleServiceImpl extends ServiceImpl<TrafficSampleMapper, TrafficSample>
        implements TrafficSampleService {
}
