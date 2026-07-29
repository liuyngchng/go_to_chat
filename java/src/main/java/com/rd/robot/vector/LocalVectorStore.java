package com.rd.robot.vector;

import com.rd.robot.model.SearchResult;
import com.rd.robot.model.VectorRecord;

import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.sql.*;
import java.util.*;
import java.util.concurrent.locks.ReadWriteLock;
import java.util.concurrent.locks.ReentrantReadWriteLock;

/**
 * Local SQLite-based vector store.
 * Stores vectors as binary BLOBs with in-memory cache for fast search.
 */
public class LocalVectorStore implements VectorStore {

    private final Connection conn;
    private final long vdbId;
    private final ReadWriteLock lock = new ReentrantReadWriteLock();
    private final List<VectorDoc> docs = new ArrayList<>();
    private int dim;

    private static class VectorDoc {
        String id;
        String content;
        double[] vector;
        String source;
    }

    public LocalVectorStore(String dbPath, long vdbId) throws SQLException {
        this.vdbId = vdbId;
        this.conn = DriverManager.getConnection("jdbc:sqlite:" + dbPath);

        try (Statement stmt = conn.createStatement()) {
            stmt.execute("PRAGMA journal_mode=WAL");
        }

        conn.setAutoCommit(true);
        migrate();
        loadMem();
    }

    private void migrate() throws SQLException {
        try (Statement stmt = conn.createStatement()) {
            stmt.execute("""
                    CREATE TABLE IF NOT EXISTS vectors (
                        id      TEXT NOT NULL,
                        vdb_id  INTEGER NOT NULL,
                        content TEXT NOT NULL,
                        vector  BLOB NOT NULL,
                        source  TEXT NOT NULL DEFAULT '',
                        PRIMARY KEY (vdb_id, id)
                    );
                    CREATE INDEX IF NOT EXISTS idx_vectors_vdb_id ON vectors(vdb_id);
                    CREATE INDEX IF NOT EXISTS idx_vectors_source ON vectors(vdb_id, source);
                    """);
        }
    }

    @Override
    public void ensureCollection(int dimension) {
        lock.writeLock().lock();
        try {
            this.dim = dimension;
        } finally {
            lock.writeLock().unlock();
        }
    }

    @Override
    public void insert(List<VectorRecord> records) throws SQLException {
        if (records.isEmpty()) return;

        String sql = "INSERT OR REPLACE INTO vectors (id, vdb_id, content, vector, source) VALUES (?, ?, ?, ?, ?)";
        try (PreparedStatement ps = conn.prepareStatement(sql)) {
            conn.setAutoCommit(false);
            try {
                List<VectorDoc> newDocs = new ArrayList<>();
                for (VectorRecord r : records) {
                    String source = r.getMeta() != null ? r.getMeta().getOrDefault("source", "") : "";
                    byte[] vecBytes = floatsToBytes(r.getVector());

                    ps.setString(1, r.getId());
                    ps.setLong(2, vdbId);
                    ps.setString(3, r.getContent());
                    ps.setBytes(4, vecBytes);
                    ps.setString(5, source);
                    ps.executeUpdate();

                    VectorDoc doc = new VectorDoc();
                    doc.id = r.getId();
                    doc.content = r.getContent();
                    doc.vector = r.getVector();
                    doc.source = source;
                    newDocs.add(doc);
                }
                conn.commit();

                // Update in-memory cache
                lock.writeLock().lock();
                try {
                    if (dim == 0 && !newDocs.isEmpty() && newDocs.get(0).vector.length > 0) {
                        dim = newDocs.get(0).vector.length;
                    }
                    Map<String, Integer> index = new HashMap<>();
                    for (int i = 0; i < docs.size(); i++) {
                        index.put(docs.get(i).id, i);
                    }
                    for (VectorDoc nd : newDocs) {
                        Integer pos = index.get(nd.id);
                        if (pos != null) {
                            docs.set(pos, nd);
                        } else {
                            docs.add(nd);
                            index.put(nd.id, docs.size() - 1);
                        }
                    }
                } finally {
                    lock.writeLock().unlock();
                }
            } catch (SQLException e) {
                conn.rollback();
                throw e;
            } finally {
                conn.setAutoCommit(true);
            }
        }
    }

    @Override
    public List<SearchResult> search(double[] queryVector, int topK, double scoreThreshold) {
        lock.readLock().lock();
        List<VectorDoc> snapshot;
        try {
            snapshot = new ArrayList<>(docs);
        } finally {
            lock.readLock().unlock();
        }

        if (snapshot.isEmpty()) return Collections.emptyList();

        List<ScoredDoc> scored = new ArrayList<>();
        for (VectorDoc doc : snapshot) {
            if (doc.vector.length != queryVector.length) continue;
            double score = cosineSimilarity(queryVector, doc.vector);
            if (score >= scoreThreshold) {
                ScoredDoc sd = new ScoredDoc();
                sd.doc = doc;
                sd.score = score;
                scored.add(sd);
            }
        }

        scored.sort((a, b) -> Double.compare(b.score, a.score));

        int resultSize = Math.min(topK, scored.size());
        List<SearchResult> results = new ArrayList<>(resultSize);
        for (int i = 0; i < resultSize; i++) {
            ScoredDoc sd = scored.get(i);
            SearchResult sr = new SearchResult();
            sr.setId(sd.doc.id);
            sr.setContent(sd.doc.content);
            sr.setMetadata(Map.of("source", sd.doc.source));
            sr.setScore(sd.score);
            results.add(sr);
        }

        return results;
    }

    @Override
    public void deleteByIds(List<String> ids) throws SQLException {
        if (ids.isEmpty()) return;

        int batchSize = 500;
        for (int i = 0; i < ids.size(); i += batchSize) {
            int end = Math.min(i + batchSize, ids.size());
            List<String> batch = ids.subList(i, end);

            StringBuilder placeholders = new StringBuilder();
            for (int j = 0; j < batch.size(); j++) {
                if (j > 0) placeholders.append(',');
                placeholders.append('?');
            }

            String sql = "DELETE FROM vectors WHERE vdb_id = ? AND id IN (" + placeholders + ")";
            try (PreparedStatement ps = conn.prepareStatement(sql)) {
                ps.setLong(1, vdbId);
                for (int j = 0; j < batch.size(); j++) {
                    ps.setString(j + 2, batch.get(j));
                }
                ps.executeUpdate();
            }
        }

        lock.writeLock().lock();
        try {
            Set<String> idSet = new HashSet<>(ids);
            docs.removeIf(d -> idSet.contains(d.id));
        } finally {
            lock.writeLock().unlock();
        }
    }

    @Override
    public void deleteBySource(String source) throws SQLException {
        String sql = "DELETE FROM vectors WHERE vdb_id = ? AND source = ?";
        try (PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setLong(1, vdbId);
            ps.setString(2, source);
            ps.executeUpdate();
        }

        lock.writeLock().lock();
        try {
            docs.removeIf(d -> source.equals(d.source));
        } finally {
            lock.writeLock().unlock();
        }
    }

    @Override
    public void purge() throws SQLException {
        String sql = "DELETE FROM vectors WHERE vdb_id = ?";
        try (PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setLong(1, vdbId);
            ps.executeUpdate();
        }

        lock.writeLock().lock();
        try {
            docs.clear();
        } finally {
            lock.writeLock().unlock();
        }
    }

    @Override
    public void close() throws Exception {
        conn.close();
    }

    // ============================================================
    // Internal methods
    // ============================================================

    private void loadMem() throws SQLException {
        String sql = "SELECT id, content, vector, source FROM vectors WHERE vdb_id = ?";
        try (PreparedStatement ps = conn.prepareStatement(sql)) {
            ps.setLong(1, vdbId);
            try (ResultSet rs = ps.executeQuery()) {
                List<VectorDoc> loaded = new ArrayList<>();
                while (rs.next()) {
                    VectorDoc doc = new VectorDoc();
                    doc.id = rs.getString("id");
                    doc.content = rs.getString("content");
                    doc.vector = bytesToFloats(rs.getBytes("vector"));
                    doc.source = rs.getString("source");
                    loaded.add(doc);
                }

                lock.writeLock().lock();
                try {
                    docs.clear();
                    docs.addAll(loaded);
                    if (!loaded.isEmpty() && dim == 0) {
                        dim = loaded.get(0).vector.length;
                    }
                } finally {
                    lock.writeLock().unlock();
                }
            }
        }
    }

    private static byte[] floatsToBytes(double[] f) {
        ByteBuffer buf = ByteBuffer.allocate(f.length * 8).order(ByteOrder.LITTLE_ENDIAN);
        for (double v : f) {
            buf.putDouble(v);
        }
        return buf.array();
    }

    private static double[] bytesToFloats(byte[] b) {
        if (b == null || b.length % 8 != 0) return new double[0];
        ByteBuffer buf = ByteBuffer.wrap(b).order(ByteOrder.LITTLE_ENDIAN);
        double[] f = new double[b.length / 8];
        for (int i = 0; i < f.length; i++) {
            f[i] = buf.getDouble();
        }
        return f;
    }

    private static double cosineSimilarity(double[] a, double[] b) {
        if (a.length != b.length || a.length == 0) return 0;

        double dotProduct = 0, normA = 0, normB = 0;
        for (int i = 0; i < a.length; i++) {
            dotProduct += a[i] * b[i];
            normA += a[i] * a[i];
            normB += b[i] * b[i];
        }

        if (normA == 0 || normB == 0) return 0;

        return dotProduct / (Math.sqrt(normA) * Math.sqrt(normB));
    }

    private static class ScoredDoc {
        VectorDoc doc;
        double score;
    }
}