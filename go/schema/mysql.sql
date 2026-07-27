-- ============================================================
-- go_to_chat 数据库表结构 (MySQL 版本)
-- ============================================================

-- 知识库元数据表
CREATE TABLE IF NOT EXISTS vdb_info (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    uid VARCHAR(128) NOT NULL DEFAULT '',
    is_public TINYINT(1) NOT NULL DEFAULT 0,
    is_default TINYINT(1) NOT NULL DEFAULT 0,
    create_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_vdb_info_uid (uid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 知识库文件信息表
CREATE TABLE IF NOT EXISTS vdb_file_info (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(512) NOT NULL,
    uid VARCHAR(128) NOT NULL DEFAULT '',
    vdb_id BIGINT NOT NULL,
    task_id VARCHAR(64) NOT NULL DEFAULT '',
    file_path VARCHAR(1024) NOT NULL DEFAULT '',
    percent DOUBLE NOT NULL DEFAULT 0,
    process_info VARCHAR(1024) NOT NULL DEFAULT '',
    file_md5 VARCHAR(64) NOT NULL DEFAULT '',
    create_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_vdb_file_info_vdb_id (vdb_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 提示词模板表
CREATE TABLE IF NOT EXISTS prompt_template (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(128) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    uid BIGINT NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 系统配置表 (key-value 存储)
CREATE TABLE IF NOT EXISTS sys_config (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    config_key VARCHAR(128) NOT NULL UNIQUE,
    config_value TEXT NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_sys_config_key (config_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    uid BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_name VARCHAR(128) NOT NULL UNIQUE,
    user_pwd VARCHAR(64) NOT NULL DEFAULT '',
    role INT NOT NULL DEFAULT 0,
    note VARCHAR(512) NOT NULL DEFAULT '',
    INDEX idx_users_name (user_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- API Token 表
CREATE TABLE IF NOT EXISTS api_tokens (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_name VARCHAR(128) NOT NULL,
    token_preview VARCHAR(32) NOT NULL DEFAULT '',
    expires_at DATETIME NOT NULL,
    create_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_api_tokens_user (user_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- API 调用日志表
CREATE TABLE IF NOT EXISTS api_call_log (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_name VARCHAR(128) NOT NULL,
    api_path VARCHAR(512) NOT NULL DEFAULT '',
    method VARCHAR(10) NOT NULL DEFAULT '',
    request_body TEXT NOT NULL,
    response_body TEXT NOT NULL,
    status_code INT NOT NULL DEFAULT 200,
    error_msg VARCHAR(1024) NOT NULL DEFAULT '',
    create_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_api_call_log_user (user_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
