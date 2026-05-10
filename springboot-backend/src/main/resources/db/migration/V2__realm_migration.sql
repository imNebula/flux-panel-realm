-- Flux Panel Realm migration.
-- This migration is additive: it keeps legacy Gost columns/routes while moving runtime state to Realm.

ALTER TABLE `node`
  ADD COLUMN `agent_version` varchar(100) DEFAULT NULL,
  ADD COLUMN `realm_version` varchar(200) DEFAULT NULL,
  ADD COLUMN `realm_binary_path` varchar(500) DEFAULT NULL,
  ADD COLUMN `realm_config_dir` varchar(500) DEFAULT NULL,
  ADD COLUMN `realm_process_name` varchar(200) DEFAULT 'flux-realm',
  ADD COLUMN `realm_service_name` varchar(200) DEFAULT 'flux-realm-agent',
  ADD COLUMN `agent_process_name` varchar(200) DEFAULT 'flux-realm-agent',
  ADD COLUMN `instance_name` varchar(200) DEFAULT 'default',
  ADD COLUMN `os` varchar(80) DEFAULT NULL,
  ADD COLUMN `distro` varchar(120) DEFAULT NULL,
  ADD COLUMN `os_version` varchar(120) DEFAULT NULL,
  ADD COLUMN `arch` varchar(80) DEFAULT NULL,
  ADD COLUMN `libc` varchar(80) DEFAULT NULL,
  ADD COLUMN `init_system` varchar(80) DEFAULT NULL,
  ADD COLUMN `container_type` varchar(80) DEFAULT NULL,
  ADD COLUMN `capabilities_json` json DEFAULT NULL,
  ADD COLUMN `running_processes_json` json DEFAULT NULL,
  ADD COLUMN `config_hash` varchar(128) DEFAULT NULL,
  ADD COLUMN `endpoint_count` int(10) DEFAULT 0,
  ADD COLUMN `active_forward_count` int(10) DEFAULT 0,
  ADD COLUMN `active_tunnel_count` int(10) DEFAULT 0,
  ADD COLUMN `last_apply_id` varchar(100) DEFAULT NULL,
  ADD COLUMN `last_apply_status` int(10) DEFAULT NULL,
  ADD COLUMN `last_apply_error` text DEFAULT NULL,
  ADD COLUMN `last_apply_json` json DEFAULT NULL;

ALTER TABLE `tunnel`
  ADD COLUMN `listen_transport` varchar(120) DEFAULT NULL,
  ADD COLUMN `remote_transport` varchar(120) DEFAULT NULL,
  ADD COLUMN `proxy_protocol` varchar(20) DEFAULT NULL,
  ADD COLUMN `process_mode` varchar(40) DEFAULT 'single',
  ADD COLUMN `process_name` varchar(200) DEFAULT NULL,
  ADD COLUMN `stats_method` varchar(40) DEFAULT 'auto',
  ADD COLUMN `latency_sample_enabled` int(10) DEFAULT 0,
  ADD COLUMN `latency_sample_interval` int(10) DEFAULT 60,
  ADD COLUMN `latency_probe_mode` varchar(40) DEFAULT 'tcp_connect';

ALTER TABLE `forward`
  ADD COLUMN `listen_transport` varchar(120) DEFAULT NULL,
  ADD COLUMN `remote_transport` varchar(120) DEFAULT NULL,
  ADD COLUMN `proxy_protocol` varchar(20) DEFAULT NULL,
  ADD COLUMN `balance_strategy` varchar(40) DEFAULT NULL,
  ADD COLUMN `extra_remotes` text DEFAULT NULL,
  ADD COLUMN `through_addr` varchar(200) DEFAULT NULL,
  ADD COLUMN `process_mode` varchar(40) DEFAULT 'single',
  ADD COLUMN `process_name` varchar(200) DEFAULT NULL,
  ADD COLUMN `stats_method` varchar(40) DEFAULT 'auto',
  ADD COLUMN `latency_sample_enabled` int(10) DEFAULT 0,
  ADD COLUMN `latency_sample_interval` int(10) DEFAULT 60;

CREATE TABLE IF NOT EXISTS `agent_capability` (
  `id` int(10) NOT NULL AUTO_INCREMENT,
  `node_id` int(10) NOT NULL,
  `capabilities_json` json NOT NULL,
  `environment_json` json DEFAULT NULL,
  `traffic_stats_method` varchar(40) DEFAULT NULL,
  `traffic_stats_reason` text DEFAULT NULL,
  `created_time` bigint(20) NOT NULL,
  `updated_time` bigint(20) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_agent_capability_node` (`node_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `apply_history` (
  `id` int(10) NOT NULL AUTO_INCREMENT,
  `node_id` int(10) NOT NULL,
  `apply_id` varchar(100) NOT NULL,
  `changed_resources` text DEFAULT NULL,
  `config_hash_before` varchar(128) DEFAULT NULL,
  `config_hash_after` varchar(128) DEFAULT NULL,
  `validation_result` text DEFAULT NULL,
  `action` varchar(40) DEFAULT NULL,
  `duration_ms` bigint(20) DEFAULT NULL,
  `success` int(10) NOT NULL DEFAULT 0,
  `error_message` text DEFAULT NULL,
  `created_time` bigint(20) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_apply_history_node_time` (`node_id`, `created_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `traffic_sample` (
  `id` int(10) NOT NULL AUTO_INCREMENT,
  `node_id` int(10) NOT NULL,
  `tunnel_id` int(10) DEFAULT NULL,
  `forward_id` int(10) DEFAULT NULL,
  `user_id` int(10) DEFAULT NULL,
  `listen_addr` varchar(200) DEFAULT NULL,
  `listen_port` int(10) DEFAULT NULL,
  `protocol` varchar(10) DEFAULT NULL,
  `in_bytes` bigint(20) NOT NULL DEFAULT 0,
  `out_bytes` bigint(20) NOT NULL DEFAULT 0,
  `total_bytes` bigint(20) NOT NULL DEFAULT 0,
  `billing_bytes` bigint(20) NOT NULL DEFAULT 0,
  `sample_time` bigint(20) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_traffic_sample_forward_time` (`forward_id`, `sample_time`),
  KEY `idx_traffic_sample_node_time` (`node_id`, `sample_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `latency_sample` (
  `id` int(10) NOT NULL AUTO_INCREMENT,
  `node_id` int(10) NOT NULL,
  `tunnel_id` int(10) DEFAULT NULL,
  `forward_id` int(10) DEFAULT NULL,
  `protocol` varchar(20) NOT NULL,
  `probe_mode` varchar(40) NOT NULL,
  `target` varchar(300) NOT NULL,
  `success` int(10) NOT NULL DEFAULT 0,
  `latency_ms` decimal(10,2) DEFAULT NULL,
  `jitter_ms` decimal(10,2) DEFAULT NULL,
  `error` text DEFAULT NULL,
  `sampled_at` bigint(20) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_latency_sample_node_time` (`node_id`, `sampled_at`),
  KEY `idx_latency_sample_forward_time` (`forward_id`, `sampled_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `latency_aggregate` (
  `id` int(10) NOT NULL AUTO_INCREMENT,
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
  `sample_count` int(10) NOT NULL DEFAULT 0,
  `created_at` bigint(20) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_latency_aggregate_scope` (`scope_type`, `scope_id`, `window`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

UPDATE `tunnel`
SET `remote_transport` = CASE
  WHEN `protocol` IN ('tls', 'ws', 'wss') THEN `protocol`
  ELSE NULL
END,
`process_mode` = 'single',
`stats_method` = 'auto',
`latency_probe_mode` = 'tcp_connect';

UPDATE `forward`
SET `balance_strategy` = CASE
  WHEN `strategy` IN ('round', 'roundrobin') THEN 'roundrobin'
  WHEN `strategy` IN ('hash', 'iphash') THEN 'iphash'
  ELSE NULL
END,
`process_mode` = 'single',
`stats_method` = 'auto';
