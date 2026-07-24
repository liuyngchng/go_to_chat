package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go_to_chat/internal/model"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore SQLite 元数据存储
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore 创建或打开 SQLite 数据库
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建数据库目录失败: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 连接池配置
	db.SetMaxOpenConns(1) // SQLite 单写
	db.SetMaxIdleConns(1)
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
