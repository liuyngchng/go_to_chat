-- ============================================================
-- go_to_chat 数据库表结构 (SQLite 版本)
-- ============================================================

-- 知识库元数据表
CREATE TABLE IF NOT EXISTS vdb_info (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    uid TEXT NOT NULL DEFAULT '',
    is_public INTEGER NOT NULL DEFAULT 0,
    is_default INTEGER NOT NULL DEFAULT 0,
    create_time DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 知识库文件信息表
CREATE TABLE IF NOT EXISTS vdb_file_info (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    uid TEXT NOT NULL DEFAULT '',
    vdb_id INTEGER NOT NULL,
    task_id TEXT NOT NULL DEFAULT '',
    file_path TEXT NOT NULL DEFAULT '',
    percent REAL NOT NULL DEFAULT 0,
    process_info TEXT NOT NULL DEFAULT '',
    file_md5 TEXT NOT NULL DEFAULT '',
    create_time DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 提示词模板表
CREATE TABLE IF NOT EXISTS prompt_template (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL,
    uid INTEGER NOT NULL DEFAULT 0
);

-- 系统配置表 (key-value 存储)
CREATE TABLE IF NOT EXISTS sys_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    config_key TEXT NOT NULL UNIQUE,
    config_value TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_vdb_info_uid ON vdb_info(uid);
CREATE INDEX IF NOT EXISTS idx_vdb_file_info_vdb_id ON vdb_file_info(vdb_id);
CREATE INDEX IF NOT EXISTS idx_sys_config_key ON sys_config(config_key);
