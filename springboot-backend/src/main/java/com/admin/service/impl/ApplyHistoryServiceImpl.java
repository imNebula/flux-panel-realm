package com.admin.service.impl;

import com.admin.entity.ApplyHistory;
import com.admin.mapper.ApplyHistoryMapper;
import com.admin.service.ApplyHistoryService;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import org.springframework.stereotype.Service;

@Service
public class ApplyHistoryServiceImpl extends ServiceImpl<ApplyHistoryMapper, ApplyHistory> implements ApplyHistoryService {
}
