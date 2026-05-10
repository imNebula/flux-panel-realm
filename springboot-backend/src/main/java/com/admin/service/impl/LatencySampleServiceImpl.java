package com.admin.service.impl;

import com.admin.entity.LatencySample;
import com.admin.mapper.LatencySampleMapper;
import com.admin.service.LatencySampleService;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import org.springframework.stereotype.Service;

@Service
public class LatencySampleServiceImpl extends ServiceImpl<LatencySampleMapper, LatencySample> implements LatencySampleService {
}
