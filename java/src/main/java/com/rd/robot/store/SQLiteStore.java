package com.rd.robot.store;

import com.alibaba.druid.pool.DruidDataSource;
import com.rd.robot.model.VdbFileInfo;
import com.rd.robot.model.VdbInfo;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.sql.*;
import java.util.ArrayList;
import java.util.List;

public class SQLiteStore {

    private static final Logger log = LoggerFactory.getLogger(SQLiteStore.class);

    private final DruidDataSource ds;

    public SQLiteStore(String dbPath) {
        // 检查数据库文件必须存在
        File dbFile = new File(dbPath);
        if (!dbFile.exists()) {
            throw new RuntimeException("数据库文件 " + dbPath + " 不存在，请从 cfg.db.template 复制");
        }

        ds = new DruidDataSource();
        ds.setUrl("jdbc:sqlite:" + dbPath);
        ds.setMaxActive(4);
        ds.setMinIdle(0);
        ds.setMaxWait(5000);
        ds.setTestOnBorrow(true);
        ds.setValidationQuery("SELECT 1");

        // 启用 WAL 模式
        try (Connection conn = ds.getConnection();
             Statement stmt = conn.createStatement()) {
            stmt.execute("PRAGMA journal_mode=WAL");
        } catch (SQLException e) {
            close();
            throw new RuntimeException("启用 WAL 失败", e);
        }

        migrate();
        log.info("SQLite 数据库初始化完成");
    }

    // ============================================================
    // 表迁移
    // ============================================================

    private void migrate() {
        String schema = """
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
                """;
        try (Connection conn = ds.getConnection();
             Statement stmt = conn.createStatement()) {
            stmt.executeUpdate(schema);
        } catch (SQLException e) {
            throw new RuntimeException("数据库迁移失败", e);
        }
    }

    // ============================================================
    // 知识库 (vdb_info) CRUD
    // ============================================================

    public long createVdb(String name, String uid, boolean isPublic) {
        String sql = "INSERT INTO vdb_info (name, uid, is_public) VALUES (?, ?, ?)";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setString(1, name);
            ps.setString(2, uid);
            ps.setInt(3, isPublic ? 1 : 0);
            ps.executeUpdate();
            try (ResultSet rs = ps.getGeneratedKeys()) {
                if (rs.next()) {
                    return rs.getLong(1);
                }
            }
            return 0;
        } catch (SQLException e) {
            throw new RuntimeException("创建知识库失败", e);
        }
    }

    public VdbInfo getVdbByID(long id) {
        String sql = "SELECT id, name, uid, is_public, is_default, create_time FROM vdb_info WHERE id = ?";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setLong(1, id);
            try (ResultSet rs = ps.executeQuery()) {
                if (rs.next()) {
                    return scanVdbInfo(rs);
                }
            }
        } catch (SQLException e) {
            throw new RuntimeException("查询知识库失败", e);
        }
        return null;
    }

    public List<VdbInfo> getUserVdbs(String uid) {
        String sql = "SELECT id, name, uid, is_public, is_default, create_time FROM vdb_info WHERE uid = ? ORDER BY create_time DESC";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setString(1, uid);
            try (ResultSet rs = ps.executeQuery()) {
                List<VdbInfo> list = new ArrayList<>();
                while (rs.next()) {
                    list.add(scanVdbInfo(rs));
                }
                return list;
            }
        } catch (SQLException e) {
            throw new RuntimeException("查询知识库列表失败", e);
        }
    }

    public List<VdbInfo> getPublicVdbs(String excludeUID) {
        String sql = "SELECT id, name, uid, is_public, is_default, create_time FROM vdb_info WHERE is_public = 1 AND uid != ? ORDER BY create_time DESC";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setString(1, excludeUID);
            try (ResultSet rs = ps.executeQuery()) {
                List<VdbInfo> list = new ArrayList<>();
                while (rs.next()) {
                    list.add(scanVdbInfo(rs));
                }
                return list;
            }
        } catch (SQLException e) {
            throw new RuntimeException("查询公开知识库失败", e);
        }
    }

    public void deleteVdb(long id) {
        try (Connection conn = ds.getConnection()) {
            conn.setAutoCommit(false);
            try {
                // 先删文件记录
                try (PreparedStatement ps = conn.prepareStatement("DELETE FROM vdb_file_info WHERE vdb_id = ?")) {
                    ps.setLong(1, id);
                    ps.executeUpdate();
                }
                // 再删知识库
                try (PreparedStatement ps = conn.prepareStatement("DELETE FROM vdb_info WHERE id = ?")) {
                    ps.setLong(1, id);
                    ps.executeUpdate();
                }
                conn.commit();
            } catch (SQLException e) {
                conn.rollback();
                throw e;
            } finally {
                conn.setAutoCommit(true);
            }
        } catch (SQLException e) {
            throw new RuntimeException("删除知识库失败", e);
        }
    }

    public void setDefaultVdb(long id, String uid) {
        try (Connection conn = ds.getConnection()) {
            conn.setAutoCommit(false);
            try {
                // 先取消该用户所有默认
                try (PreparedStatement ps = conn.prepareStatement("UPDATE vdb_info SET is_default = 0 WHERE uid = ?")) {
                    ps.setString(1, uid);
                    ps.executeUpdate();
                }
                // 设置新的默认
                try (PreparedStatement ps = conn.prepareStatement("UPDATE vdb_info SET is_default = 1 WHERE id = ? AND uid = ?")) {
                    ps.setLong(1, id);
                    ps.setString(2, uid);
                    ps.executeUpdate();
                }
                conn.commit();
            } catch (SQLException e) {
                conn.rollback();
                throw e;
            } finally {
                conn.setAutoCommit(true);
            }
        } catch (SQLException e) {
            throw new RuntimeException("设置默认知识库失败", e);
        }
    }

    public boolean checkVdbNameExists(String name, String uid) {
        String sql = "SELECT COUNT(*) FROM vdb_info WHERE name = ? AND uid = ?";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setString(1, name);
            ps.setString(2, uid);
            try (ResultSet rs = ps.executeQuery()) {
                if (rs.next()) {
                    return rs.getInt(1) > 0;
                }
            }
        } catch (SQLException e) {
            throw new RuntimeException("检查知识库名称失败", e);
        }
        return false;
    }

    public long getDefaultVdbID(String uid) {
        String sql = "SELECT id FROM vdb_info WHERE uid = ? AND is_default = 1 LIMIT 1";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setString(1, uid);
            try (ResultSet rs = ps.executeQuery()) {
                if (rs.next()) {
                    return rs.getLong(1);
                }
            }
        } catch (SQLException e) {
            throw new RuntimeException("查询默认知识库失败", e);
        }
        return 0;
    }

    // ============================================================
    // 文件 (vdb_file_info) CRUD
    // ============================================================

    public long createFileInfo(VdbFileInfo info) {
        String sql = "INSERT INTO vdb_file_info (name, uid, vdb_id, task_id, file_path, percent, process_info, file_md5) VALUES (?, ?, ?, ?, ?, ?, ?, ?)";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setString(1, info.getName());
            ps.setString(2, info.getUid());
            ps.setLong(3, info.getVdbId());
            ps.setString(4, info.getTaskId());
            ps.setString(5, info.getFilePath());
            ps.setDouble(6, info.getPercent());
            ps.setString(7, info.getProcessInfo());
            ps.setString(8, info.getFileMd5());
            ps.executeUpdate();
            try (ResultSet rs = ps.getGeneratedKeys()) {
                if (rs.next()) {
                    return rs.getLong(1);
                }
            }
            return 0;
        } catch (SQLException e) {
            throw new RuntimeException("创建文件记录失败", e);
        }
    }

    public List<VdbFileInfo> getFilesByVdbID(long vdbId) {
        String sql = "SELECT id, name, uid, vdb_id, task_id, file_path, percent, process_info, file_md5, create_time FROM vdb_file_info WHERE vdb_id = ? ORDER BY create_time DESC";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setLong(1, vdbId);
            try (ResultSet rs = ps.executeQuery()) {
                List<VdbFileInfo> list = new ArrayList<>();
                while (rs.next()) {
                    list.add(scanVdbFileInfo(rs));
                }
                return list;
            }
        } catch (SQLException e) {
            throw new RuntimeException("查询文件列表失败", e);
        }
    }

    public VdbFileInfo getFileByID(long id) {
        String sql = "SELECT id, name, uid, vdb_id, task_id, file_path, percent, process_info, file_md5, create_time FROM vdb_file_info WHERE id = ?";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setLong(1, id);
            try (ResultSet rs = ps.executeQuery()) {
                if (rs.next()) {
                    return scanVdbFileInfo(rs);
                }
            }
        } catch (SQLException e) {
            throw new RuntimeException("查询文件失败", e);
        }
        return null;
    }

    public List<VdbFileInfo> getUnprocessedFiles() {
        String sql = "SELECT id, name, uid, vdb_id, task_id, file_path, percent, process_info, file_md5, create_time FROM vdb_file_info WHERE percent != 100 ORDER BY create_time ASC";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            try (ResultSet rs = ps.executeQuery()) {
                List<VdbFileInfo> list = new ArrayList<>();
                while (rs.next()) {
                    list.add(scanVdbFileInfo(rs));
                }
                return list;
            }
        } catch (SQLException e) {
            throw new RuntimeException("查询未处理文件失败", e);
        }
    }

    public void updateFileProgress(long id, double percent, String info) {
        String sql = "UPDATE vdb_file_info SET percent = ?, process_info = ? WHERE id = ?";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setDouble(1, percent);
            ps.setString(2, info);
            ps.setLong(3, id);
            ps.executeUpdate();
        } catch (SQLException e) {
            throw new RuntimeException("更新文件进度失败", e);
        }
    }

    public void deleteFile(long id) {
        String sql = "DELETE FROM vdb_file_info WHERE id = ?";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setLong(1, id);
            ps.executeUpdate();
        } catch (SQLException e) {
            throw new RuntimeException("删除文件记录失败", e);
        }
    }

    public VdbFileInfo checkFileMD5Exists(long vdbId, String md5) {
        String sql = "SELECT id, name, uid, vdb_id, task_id, file_path, percent, process_info, file_md5, create_time FROM vdb_file_info WHERE vdb_id = ? AND file_md5 = ?";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setLong(1, vdbId);
            ps.setString(2, md5);
            try (ResultSet rs = ps.executeQuery()) {
                if (rs.next()) {
                    return scanVdbFileInfo(rs);
                }
            }
        } catch (SQLException e) {
            throw new RuntimeException("检查文件MD5失败", e);
        }
        return null;
    }

    // ============================================================
    // 提示词模板
    // ============================================================

    public String getPrompt(String name) {
        String sql = "SELECT value FROM prompt_template WHERE name = ?";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setString(1, name);
            try (ResultSet rs = ps.executeQuery()) {
                if (rs.next()) {
                    return rs.getString("value");
                }
            }
        } catch (SQLException e) {
            throw new RuntimeException("查询提示词失败", e);
        }
        return "";
    }

    public void upsertPrompt(String name, String value, int uid) {
        String sql = "INSERT INTO prompt_template (name, value, uid) VALUES (?, ?, ?) ON CONFLICT(name) DO UPDATE SET value = excluded.value, uid = excluded.uid";
        try (Connection conn = ds.getConnection();
             PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setString(1, name);
            ps.setString(2, value);
            ps.setInt(3, uid);
            ps.executeUpdate();
        } catch (SQLException e) {
            throw new RuntimeException("保存提示词失败", e);
        }
    }

    // ============================================================
    // 关闭
    // ============================================================

    public void close() {
        ds.close();
    }

    public Connection getConnection() throws SQLException {
        return ds.getConnection();
    }

    // ============================================================
    // 辅助方法
    // ============================================================

    private VdbInfo scanVdbInfo(ResultSet rs) throws SQLException {
        VdbInfo v = new VdbInfo();
        v.setId(rs.getLong("id"));
        v.setName(rs.getString("name"));
        v.setUid(rs.getString("uid"));
        v.setPublic(rs.getInt("is_public") != 0);
        v.setDefault(rs.getInt("is_default") != 0);
        v.setCreateTime(rs.getString("create_time"));
        return v;
    }

    private VdbFileInfo scanVdbFileInfo(ResultSet rs) throws SQLException {
        VdbFileInfo f = new VdbFileInfo();
        f.setId(rs.getLong("id"));
        f.setName(rs.getString("name"));
        f.setUid(rs.getString("uid"));
        f.setVdbId(rs.getLong("vdb_id"));
        f.setTaskId(rs.getString("task_id"));
        f.setFilePath(rs.getString("file_path"));
        f.setPercent(rs.getDouble("percent"));
        f.setProcessInfo(rs.getString("process_info"));
        f.setFileMd5(rs.getString("file_md5"));
        f.setCreateTime(rs.getString("create_time"));
        return f;
    }
}
