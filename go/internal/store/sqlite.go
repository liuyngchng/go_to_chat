package store

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"go_to_chat/internal/model"

	_ "modernc.org/sqlite"
)

// SQLiteStore SQLite 元数据存储
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore 打开 SQLite 数据库（文件必须已存在）
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// 检查数据库文件是否存在，不允许自动创建
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("数据库文件 %s 不存在，请从 cfg.db.template 复制", dbPath)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// WAL 模式：读不阻塞写
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("启用 WAL 失败: %w", err)
	}

	// 读多写少场景，允许多个并发读
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return store, nil
}

// Close 关闭数据库连接
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// migrate 创建表结构
func (s *SQLiteStore) migrate() error {
	schema := `
		CREATE TABLE IF NOT EXISTS vdb_info (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			uid TEXT NOT NULL DEFAULT '',
			is_public INTEGER NOT NULL DEFAULT 0,
			is_default INTEGER NOT NULL DEFAULT 0,
			create_time DATETIME DEFAULT CURRENT_TIMESTAMP
		);

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

		CREATE TABLE IF NOT EXISTS prompt_template (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			value TEXT NOT NULL,
			uid INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS sys_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			config_key TEXT NOT NULL UNIQUE,
			config_value TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		`
	_, err := s.db.Exec(schema)
	return err
}

// ============================================================
// 知识库 (vdb_info) CRUD
// ============================================================

// CreateVdb 创建知识库
func (s *SQLiteStore) CreateVdb(name, uid string, isPublic bool) (int64, error) {
	pubVal := 0
	if isPublic {
		pubVal = 1
	}

	result, err := s.db.Exec(
		"INSERT INTO vdb_info (name, uid, is_public) VALUES (?, ?, ?)",
		name, uid, pubVal,
	)
	if err != nil {
		return 0, fmt.Errorf("创建知识库失败: %w", err)
	}
	return result.LastInsertId()
}

// GetVdbByID 根据 ID 获取知识库
func (s *SQLiteStore) GetVdbByID(id int64) (*model.VdbInfo, error) {
	row := s.db.QueryRow(
		"SELECT id, name, uid, is_public, is_default, create_time FROM vdb_info WHERE id = ?", id,
	)
	return scanVdbInfo(row)
}

// GetUserVdbs 获取用户的所有知识库
func (s *SQLiteStore) GetUserVdbs(uid string) ([]model.VdbInfo, error) {
	rows, err := s.db.Query(
		"SELECT id, name, uid, is_public, is_default, create_time FROM vdb_info WHERE uid = ? ORDER BY create_time DESC",
		uid,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanVdbInfoList(rows)
}

// GetPublicVdbs 获取所有公开的知识库
func (s *SQLiteStore) GetPublicVdbs(excludeUID string) ([]model.VdbInfo, error) {
	rows, err := s.db.Query(
		"SELECT id, name, uid, is_public, is_default, create_time FROM vdb_info WHERE is_public = 1 AND uid != ? ORDER BY create_time DESC",
		excludeUID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanVdbInfoList(rows)
}

// DeleteVdb 删除知识库及其所有文件
func (s *SQLiteStore) DeleteVdb(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 先删文件记录
	if _, err := tx.Exec("DELETE FROM vdb_file_info WHERE vdb_id = ?", id); err != nil {
		return err
	}
	// 再删知识库
	if _, err := tx.Exec("DELETE FROM vdb_info WHERE id = ?", id); err != nil {
		return err
	}

	return tx.Commit()
}

// SetDefaultVdb 设置默认知识库
func (s *SQLiteStore) SetDefaultVdb(id int64, uid string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 先取消该用户所有默认
	if _, err := tx.Exec("UPDATE vdb_info SET is_default = 0 WHERE uid = ?", uid); err != nil {
		return err
	}
	// 设置新的默认
	if _, err := tx.Exec("UPDATE vdb_info SET is_default = 1 WHERE id = ? AND uid = ?", id, uid); err != nil {
		return err
	}

	return tx.Commit()
}

// CheckVdbNameExists 检查知识库名称是否已存在
func (s *SQLiteStore) CheckVdbNameExists(name, uid string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM vdb_info WHERE name = ? AND uid = ?", name, uid,
	).Scan(&count)
	return count > 0, err
}

// GetDefaultVdbID 获取用户的默认知识库 ID
func (s *SQLiteStore) GetDefaultVdbID(uid string) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		"SELECT id FROM vdb_info WHERE uid = ? AND is_default = 1 LIMIT 1", uid,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

// ============================================================
// 文件 (vdb_file_info) CRUD
// ============================================================

// CreateFileInfo 创建文件记录
func (s *SQLiteStore) CreateFileInfo(info *model.VdbFileInfo) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO vdb_file_info (name, uid, vdb_id, task_id, file_path, percent, process_info, file_md5)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		info.Name, info.UID, info.VdbID, info.TaskID, info.FilePath, info.Percent, info.ProcessInfo, info.FileMD5,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetFilesByVdbID 获取知识库下的所有文件
func (s *SQLiteStore) GetFilesByVdbID(vdbID int64) ([]model.VdbFileInfo, error) {
	rows, err := s.db.Query(
		`SELECT id, name, uid, vdb_id, task_id, file_path, percent, process_info, file_md5, create_time
		 FROM vdb_file_info WHERE vdb_id = ? ORDER BY create_time DESC`, vdbID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []model.VdbFileInfo
	for rows.Next() {
		var f model.VdbFileInfo
		if err := rows.Scan(&f.ID, &f.Name, &f.UID, &f.VdbID, &f.TaskID,
			&f.FilePath, &f.Percent, &f.ProcessInfo, &f.FileMD5, &f.CreateTime); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// GetFileByID 根据 ID 获取文件信息
func (s *SQLiteStore) GetFileByID(id int64) (*model.VdbFileInfo, error) {
	row := s.db.QueryRow(
		`SELECT id, name, uid, vdb_id, task_id, file_path, percent, process_info, file_md5, create_time
		 FROM vdb_file_info WHERE id = ?`, id,
	)
	var f model.VdbFileInfo
	err := row.Scan(&f.ID, &f.Name, &f.UID, &f.VdbID, &f.TaskID,
		&f.FilePath, &f.Percent, &f.ProcessInfo, &f.FileMD5, &f.CreateTime)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// GetUnprocessedFiles 获取未处理的文件
func (s *SQLiteStore) GetUnprocessedFiles() ([]model.VdbFileInfo, error) {
	rows, err := s.db.Query(
		`SELECT id, name, uid, vdb_id, task_id, file_path, percent, process_info, file_md5, create_time
		 FROM vdb_file_info WHERE percent != 100 ORDER BY create_time ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []model.VdbFileInfo
	for rows.Next() {
		var f model.VdbFileInfo
		if err := rows.Scan(&f.ID, &f.Name, &f.UID, &f.VdbID, &f.TaskID,
			&f.FilePath, &f.Percent, &f.ProcessInfo, &f.FileMD5, &f.CreateTime); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// UpdateFileProgress 更新文件处理进度
func (s *SQLiteStore) UpdateFileProgress(id int64, percent float64, info string) error {
	_, err := s.db.Exec(
		"UPDATE vdb_file_info SET percent = ?, process_info = ? WHERE id = ?",
		percent, info, id,
	)
	return err
}

// DeleteFile 删除文件记录
func (s *SQLiteStore) DeleteFile(id int64) error {
	_, err := s.db.Exec("DELETE FROM vdb_file_info WHERE id = ?", id)
	return err
}

// CheckFileMD5Exists 检查同一知识库下的文件 MD5 是否已存在
func (s *SQLiteStore) CheckFileMD5Exists(vdbID int64, md5 string) (*model.VdbFileInfo, error) {
	row := s.db.QueryRow(
		`SELECT id, name, uid, vdb_id, task_id, file_path, percent, process_info, file_md5, create_time
		 FROM vdb_file_info WHERE vdb_id = ? AND file_md5 = ?`, vdbID, md5,
	)
	var f model.VdbFileInfo
	err := row.Scan(&f.ID, &f.Name, &f.UID, &f.VdbID, &f.TaskID,
		&f.FilePath, &f.Percent, &f.ProcessInfo, &f.FileMD5, &f.CreateTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ============================================================
// 提示词模板
// ============================================================

// GetPrompt 根据名称获取提示词模板，不存在则返回空字符串
func (s *SQLiteStore) GetPrompt(name string) (string, error) {
	var value string
	err := s.db.QueryRow(
		"SELECT value FROM prompt_template WHERE name = ?", name,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// UpsertPrompt 插入或更新提示词模板
func (s *SQLiteStore) UpsertPrompt(name, value string, uid int) error {
	_, err := s.db.Exec(
		`INSERT INTO prompt_template (name, value, uid) VALUES (?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET value = excluded.value, uid = excluded.uid`,
		name, value, uid,
	)
	return err
}

// ============================================================
// 系统配置 (sys_config)
// ============================================================

// GetConfig 获取单个配置值
func (s *SQLiteStore) GetConfig(key string) (string, error) {
	var value string
	err := s.db.QueryRow(
		"SELECT config_value FROM sys_config WHERE config_key = ?", key,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetConfig 设置单个配置值（插入或更新）
func (s *SQLiteStore) SetConfig(key, value, description string) error {
	_, err := s.db.Exec(
		`INSERT INTO sys_config (config_key, config_value, description, updated_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(config_key) DO UPDATE SET
		 config_value = excluded.config_value,
		 description = excluded.description,
		 updated_at = CURRENT_TIMESTAMP`,
		key, value, description,
	)
	return err
}

// GetAllConfigs 获取所有配置项
func (s *SQLiteStore) GetAllConfigs() (map[string]string, error) {
	rows, err := s.db.Query("SELECT config_key, config_value FROM sys_config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, rows.Err()
}

// SeedDefaultConfigs 初始化默认配置（仅当 sys_config 表为空时执行）
// 从 cfg.yml 中读取的值作为初始种子
func (s *SQLiteStore) SeedDefaultConfigs(sysName, sysAuth string, apiCfg APISeedConfig) error {
	// 检查是否已有配置
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM sys_config").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // 已有配置，跳过种子
	}

	entries := []struct{ key, value, desc string }{
		{"sys.name", sysName, "系统名称"},
		{"sys.auth", sysAuth, "是否启用认证 (true/false)"},
		{"api.llm_api_uri", apiCfg.LLMAPIURI, "LLM API 地址"},
		{"api.llm_api_key", apiCfg.LLMAPIKey, "LLM API Key"},
		{"api.llm_model_name", apiCfg.LLMModelName, "LLM 模型名称"},
		{"api.embedding_api_uri", apiCfg.EmbeddingAPIURI, "Embedding API 地址"},
		{"api.embedding_api_key", apiCfg.EmbeddingAPIKey, "Embedding API Key"},
		{"api.embedding_model_name", apiCfg.EmbeddingModelName, "Embedding 模型名称"},
		{"prompt.chat_msg", defaultChatPrompt, "聊天提示词模板"},
			// 知识库参数
			{"kb.chunk_size", "300", "文本分片大小（字符数）"},
			{"kb.chunk_overlap", "80", "文本分片重叠大小（字符数）"},
			{"kb.top_k", "3", "检索返回条数"},
			{"kb.score_threshold", "0.1", "检索相似度阈值"},
			// LLM 参数
			{"llm.temperature", "0.7", "LLM 温度参数 (0-2)"},
			{"llm.top_p", "0.9", "LLM Top-P 采样参数 (0-1)"},
			{"llm.max_tokens", "2048", "LLM 最大生成 Token 数"},
	}

	for _, e := range entries {
		if err := s.SetConfig(e.key, e.value, e.desc); err != nil {
			return err
		}
	}
	return nil
}

// APISeedConfig 用于初始化 API 配置的种子数据
type APISeedConfig struct {
	LLMAPIURI          string
	LLMAPIKey          string
	LLMModelName       string
	EmbeddingAPIURI    string
	EmbeddingAPIKey    string
	EmbeddingModelName string
}

// DefaultChatPrompt 返回默认聊天提示词模板
func DefaultChatPrompt() string {
	return defaultChatPrompt
}

const defaultChatPrompt = `你是专业的对话机器人，负责解答客户咨询。你必须基于以下知识库信息回答用户问题。
如果知识库中没有相关信息，请引导用户转接人工客服。

今日日期：{cur_date}（星期{cur_week}）

知识库内容：
---
{context}
---

历史对话：
{history}

用户问题：{question}

请用亲切、专业的中文回答：`

// ============================================================
// 辅助函数
// ============================================================

func scanVdbInfo(row *sql.Row) (*model.VdbInfo, error) {
	var v model.VdbInfo
	var isPublic, isDefault int
	err := row.Scan(&v.ID, &v.Name, &v.UID, &isPublic, &isDefault, &v.CreateTime)
	if err != nil {
		return nil, err
	}
	v.IsPublic = isPublic != 0
	v.IsDefault = isDefault != 0
	return &v, nil
}

func scanVdbInfoList(rows *sql.Rows) ([]model.VdbInfo, error) {
	var list []model.VdbInfo
	for rows.Next() {
		var v model.VdbInfo
		var isPublic, isDefault int
		if err := rows.Scan(&v.ID, &v.Name, &v.UID, &isPublic, &isDefault, &v.CreateTime); err != nil {
			return nil, err
		}
		v.IsPublic = isPublic != 0
		v.IsDefault = isDefault != 0
		list = append(list, v)
	}
	return list, rows.Err()
}
