-- phpMyAdmin SQL Dump
-- version 5.2.0
-- https://www.phpmyadmin.net/
--
-- 主机： localhost
-- 生成日期： 2025-08-14 21:52:52
-- 服务器版本： 5.7.40-log
-- PHP 版本： 7.4.33

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- 数据库： `flux_realm`
--

-- --------------------------------------------------------

--
-- 表的结构 `forward`
--

CREATE TABLE `forward` (
  `id` int(10) NOT NULL,
  `user_id` int(10) NOT NULL,
  `user_name` varchar(100) NOT NULL,
  `name` varchar(100) NOT NULL,
  `tunnel_id` int(10) NOT NULL,
  `in_port` int(10) NOT NULL,
  `out_port` int(10) DEFAULT NULL,
  `remote_addr` longtext NOT NULL,
  `strategy` varchar(100) NOT NULL DEFAULT 'fifo',
  `interface_name` varchar(200) DEFAULT NULL,
  `listen_transport` varchar(120) DEFAULT NULL,
  `remote_transport` varchar(120) DEFAULT NULL,
  `proxy_protocol` varchar(20) DEFAULT NULL,
  `balance_strategy` varchar(40) DEFAULT NULL,
  `extra_remotes` text DEFAULT NULL,
  `through_addr` varchar(200) DEFAULT NULL,
  `process_mode` varchar(40) DEFAULT 'single',
  `process_name` varchar(200) DEFAULT NULL,
  `stats_method` varchar(40) DEFAULT 'auto',
  `latency_sample_enabled` int(10) DEFAULT '0',
  `latency_sample_interval` int(10) DEFAULT '60',
  `in_flow` bigint(20) NOT NULL DEFAULT '0',
  `out_flow` bigint(20) NOT NULL DEFAULT '0',
  `created_time` bigint(20) NOT NULL,
  `updated_time` bigint(20) NOT NULL,
  `status` int(10) NOT NULL,
  `inx` int(10) NOT NULL DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `node`
--

CREATE TABLE `node` (
  `id` int(10) NOT NULL,
  `name` varchar(100) NOT NULL,
  `secret` varchar(100) NOT NULL,
  `ip` longtext,
  `server_ip` varchar(100) NOT NULL,
  `port_sta` int(10) NOT NULL,
  `port_end` int(10) NOT NULL,
  `version` varchar(100) DEFAULT NULL,
  `http` int(10) NOT NULL DEFAULT '0',
  `tls` int(10) NOT NULL DEFAULT '0',
  `socks` int(10) NOT NULL DEFAULT '0',
  `agent_version` varchar(100) DEFAULT NULL,
  `realm_version` varchar(200) DEFAULT NULL,
  `realm_binary_path` varchar(500) DEFAULT NULL,
  `realm_config_dir` varchar(500) DEFAULT NULL,
  `realm_process_name` varchar(200) DEFAULT 'flux-realm',
  `realm_service_name` varchar(200) DEFAULT 'flux-realm-agent',
  `agent_process_name` varchar(200) DEFAULT 'flux-realm-agent',
  `instance_name` varchar(200) DEFAULT 'default',
  `os` varchar(80) DEFAULT NULL,
  `distro` varchar(120) DEFAULT NULL,
  `os_version` varchar(120) DEFAULT NULL,
  `arch` varchar(80) DEFAULT NULL,
  `libc` varchar(80) DEFAULT NULL,
  `init_system` varchar(80) DEFAULT NULL,
  `container_type` varchar(80) DEFAULT NULL,
  `capabilities_json` json DEFAULT NULL,
  `running_processes_json` json DEFAULT NULL,
  `config_hash` varchar(128) DEFAULT NULL,
  `endpoint_count` int(10) DEFAULT '0',
  `active_forward_count` int(10) DEFAULT '0',
  `active_tunnel_count` int(10) DEFAULT '0',
  `last_apply_id` varchar(100) DEFAULT NULL,
  `last_apply_status` int(10) DEFAULT NULL,
  `last_apply_error` text DEFAULT NULL,
  `last_apply_json` json DEFAULT NULL,
  `created_time` bigint(20) NOT NULL,
  `updated_time` bigint(20) DEFAULT NULL,
  `status` int(10) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `speed_limit`
--

CREATE TABLE `speed_limit` (
  `id` int(10) NOT NULL,
  `name` varchar(100) NOT NULL,
  `speed` int(10) NOT NULL,
  `tunnel_id` int(10) NOT NULL,
  `tunnel_name` varchar(100) NOT NULL,
  `created_time` bigint(20) NOT NULL,
  `updated_time` bigint(20) DEFAULT NULL,
  `status` int(10) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `statistics_flow`
--

CREATE TABLE `statistics_flow` (
  `id` int(10) NOT NULL,
  `user_id` int(10) NOT NULL,
  `flow` bigint(20) NOT NULL,
  `total_flow` bigint(20) NOT NULL,
  `time` varchar(100) NOT NULL,
  `created_time` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `tunnel`
--

CREATE TABLE `tunnel` (
  `id` int(10) NOT NULL,
  `name` varchar(100) NOT NULL,
  `traffic_ratio` decimal(10,1) NOT NULL DEFAULT '1.0',
  `in_node_id` int(10) NOT NULL,
  `in_ip` varchar(100) NOT NULL,
  `out_node_id` int(10) NOT NULL,
  `out_ip` varchar(100) NOT NULL,
  `type` int(10) NOT NULL,
  `protocol` varchar(10) NOT NULL DEFAULT 'tls',
  `flow` int(10) NOT NULL,
  `tcp_listen_addr` varchar(100) NOT NULL DEFAULT '[::]',
  `udp_listen_addr` varchar(100) NOT NULL DEFAULT '[::]',
  `interface_name` varchar(200) DEFAULT NULL,
  `listen_transport` varchar(120) DEFAULT NULL,
  `remote_transport` varchar(120) DEFAULT NULL,
  `proxy_protocol` varchar(20) DEFAULT NULL,
  `process_mode` varchar(40) DEFAULT 'single',
  `process_name` varchar(200) DEFAULT NULL,
  `stats_method` varchar(40) DEFAULT 'auto',
  `latency_sample_enabled` int(10) DEFAULT '0',
  `latency_sample_interval` int(10) DEFAULT '60',
  `latency_probe_mode` varchar(40) DEFAULT 'tcp_connect',
  `created_time` bigint(20) NOT NULL,
  `updated_time` bigint(20) NOT NULL,
  `status` int(10) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `user`
--

CREATE TABLE `user` (
  `id` int(10) NOT NULL,
  `user` varchar(100) NOT NULL,
  `pwd` varchar(100) NOT NULL,
  `role_id` int(10) NOT NULL,
  `exp_time` bigint(20) NOT NULL,
  `flow` bigint(20) NOT NULL,
  `in_flow` bigint(20) NOT NULL DEFAULT '0',
  `out_flow` bigint(20) NOT NULL DEFAULT '0',
  `flow_reset_time` bigint(20) NOT NULL,
  `num` int(10) NOT NULL,
  `created_time` bigint(20) NOT NULL,
  `updated_time` bigint(20) DEFAULT NULL,
  `status` int(10) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

--
-- 转存表中的数据 `user`
--

INSERT INTO `user` (`id`, `user`, `pwd`, `role_id`, `exp_time`, `flow`, `in_flow`, `out_flow`, `flow_reset_time`, `num`, `created_time`, `updated_time`, `status`) VALUES
(1, 'admin_user', '3c85cdebade1c51cf64ca9f3c09d182d', 0, 2727251700000, 99999, 0, 0, 1, 99999, 1748914865000, 1754011744252, 1);

-- --------------------------------------------------------

--
-- 表的结构 `user_tunnel`
--

CREATE TABLE `user_tunnel` (
  `id` int(10) NOT NULL,
  `user_id` int(10) NOT NULL,
  `tunnel_id` int(10) NOT NULL,
  `speed_id` int(10) DEFAULT NULL,
  `num` int(10) NOT NULL,
  `flow` bigint(20) NOT NULL,
  `in_flow` bigint(20) NOT NULL DEFAULT '0',
  `out_flow` bigint(20) NOT NULL DEFAULT '0',
  `flow_reset_time` bigint(20) NOT NULL,
  `exp_time` bigint(20) NOT NULL,
  `status` int(10) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `vite_config`
--

CREATE TABLE `vite_config` (
  `id` int(10) NOT NULL,
  `name` varchar(200) NOT NULL,
  `value` varchar(200) NOT NULL,
  `time` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

--
-- 转存表中的数据 `vite_config`
--

INSERT INTO `vite_config` (`id`, `name`, `value`, `time`) VALUES
(1, 'app_name', 'flux', 1755147963000);

-- --------------------------------------------------------

--
-- 表的结构 `agent_capability`
--

CREATE TABLE `agent_capability` (
  `id` int(10) NOT NULL,
  `node_id` int(10) NOT NULL,
  `capabilities_json` json NOT NULL,
  `environment_json` json DEFAULT NULL,
  `traffic_stats_method` varchar(40) DEFAULT NULL,
  `traffic_stats_reason` text DEFAULT NULL,
  `created_time` bigint(20) NOT NULL,
  `updated_time` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `apply_history`
--

CREATE TABLE `apply_history` (
  `id` int(10) NOT NULL,
  `node_id` int(10) NOT NULL,
  `apply_id` varchar(100) NOT NULL,
  `changed_resources` text DEFAULT NULL,
  `config_hash_before` varchar(128) DEFAULT NULL,
  `config_hash_after` varchar(128) DEFAULT NULL,
  `validation_result` text DEFAULT NULL,
  `action` varchar(40) DEFAULT NULL,
  `duration_ms` bigint(20) DEFAULT NULL,
  `success` int(10) NOT NULL DEFAULT '0',
  `error_message` text DEFAULT NULL,
  `created_time` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `traffic_sample`
--

CREATE TABLE `traffic_sample` (
  `id` int(10) NOT NULL,
  `node_id` int(10) NOT NULL,
  `tunnel_id` int(10) DEFAULT NULL,
  `forward_id` int(10) DEFAULT NULL,
  `user_id` int(10) DEFAULT NULL,
  `listen_addr` varchar(200) DEFAULT NULL,
  `listen_port` int(10) DEFAULT NULL,
  `protocol` varchar(10) DEFAULT NULL,
  `in_bytes` bigint(20) NOT NULL DEFAULT '0',
  `out_bytes` bigint(20) NOT NULL DEFAULT '0',
  `total_bytes` bigint(20) NOT NULL DEFAULT '0',
  `billing_bytes` bigint(20) NOT NULL DEFAULT '0',
  `sample_time` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `latency_sample`
--

CREATE TABLE `latency_sample` (
  `id` int(10) NOT NULL,
  `node_id` int(10) NOT NULL,
  `tunnel_id` int(10) DEFAULT NULL,
  `forward_id` int(10) DEFAULT NULL,
  `protocol` varchar(20) NOT NULL,
  `probe_mode` varchar(40) NOT NULL,
  `target` varchar(300) NOT NULL,
  `success` int(10) NOT NULL DEFAULT '0',
  `latency_ms` decimal(10,2) DEFAULT NULL,
  `jitter_ms` decimal(10,2) DEFAULT NULL,
  `error` text DEFAULT NULL,
  `sampled_at` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `latency_aggregate`
--

CREATE TABLE `latency_aggregate` (
  `id` int(10) NOT NULL,
  `scope_type` varchar(20) NOT NULL,
  `scope_id` int(10) NOT NULL,
  `window` varchar(20) NOT NULL,
  `avg_ms` decimal(10,2) DEFAULT NULL,
  `min_ms` decimal(10,2) DEFAULT NULL,
  `max_ms` decimal(10,2) DEFAULT NULL,
  `p50_ms` decimal(10,2) DEFAULT NULL,
  `p95_ms` decimal(10,2) DEFAULT NULL,
  `p99_ms` decimal(10,2) DEFAULT NULL,
  `loss_rate` decimal(6,4) DEFAULT NULL,
  `sample_count` int(10) NOT NULL DEFAULT '0',
  `created_at` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

--
-- 转储表的索引
--

--
-- 表的索引 `forward`
--
ALTER TABLE `forward`
  ADD PRIMARY KEY (`id`);

--
-- 表的索引 `node`
--
ALTER TABLE `node`
  ADD PRIMARY KEY (`id`);

--
-- 表的索引 `speed_limit`
--
ALTER TABLE `speed_limit`
  ADD PRIMARY KEY (`id`);

--
-- 表的索引 `statistics_flow`
--
ALTER TABLE `statistics_flow`
  ADD PRIMARY KEY (`id`);

--
-- 表的索引 `tunnel`
--
ALTER TABLE `tunnel`
  ADD PRIMARY KEY (`id`);

--
-- 表的索引 `user`
--
ALTER TABLE `user`
  ADD PRIMARY KEY (`id`);

--
-- 表的索引 `user_tunnel`
--
ALTER TABLE `user_tunnel`
  ADD PRIMARY KEY (`id`);

--
-- 表的索引 `vite_config`
--
ALTER TABLE `vite_config`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `name` (`name`);

ALTER TABLE `agent_capability`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `uniq_agent_capability_node` (`node_id`);

ALTER TABLE `apply_history`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_apply_history_node_time` (`node_id`, `created_time`);

ALTER TABLE `traffic_sample`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_traffic_sample_forward_time` (`forward_id`, `sample_time`),
  ADD KEY `idx_traffic_sample_node_time` (`node_id`, `sample_time`);

ALTER TABLE `latency_sample`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_latency_sample_node_time` (`node_id`, `sampled_at`),
  ADD KEY `idx_latency_sample_forward_time` (`forward_id`, `sampled_at`);

ALTER TABLE `latency_aggregate`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_latency_aggregate_scope` (`scope_type`, `scope_id`, `window`, `created_at`);

--
-- 在导出的表使用AUTO_INCREMENT
--

--
-- 使用表AUTO_INCREMENT `forward`
--
ALTER TABLE `forward`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;

--
-- 使用表AUTO_INCREMENT `node`
--
ALTER TABLE `node`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;

--
-- 使用表AUTO_INCREMENT `speed_limit`
--
ALTER TABLE `speed_limit`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;

--
-- 使用表AUTO_INCREMENT `statistics_flow`
--
ALTER TABLE `statistics_flow`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;

--
-- 使用表AUTO_INCREMENT `tunnel`
--
ALTER TABLE `tunnel`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;

--
-- 使用表AUTO_INCREMENT `user`
--
ALTER TABLE `user`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;

--
-- 使用表AUTO_INCREMENT `user_tunnel`
--
ALTER TABLE `user_tunnel`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;

--
-- 使用表AUTO_INCREMENT `vite_config`
--
ALTER TABLE `vite_config`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;

ALTER TABLE `agent_capability`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;

ALTER TABLE `apply_history`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;

ALTER TABLE `traffic_sample`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;

ALTER TABLE `latency_sample`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;

ALTER TABLE `latency_aggregate`
  MODIFY `id` int(10) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=1;
COMMIT;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
