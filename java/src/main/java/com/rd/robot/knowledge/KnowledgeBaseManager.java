package com.rd.robot.knowledge;

import com.rd.robot.client.EmbeddingClient;
import com.rd.robot.client.RerankClient;
import com.rd.robot.model.*;
import com.rd.robot.repository.MetaStore;
import com.rd.robot.vector.VectorStore;
import com.rd.robot.vector.VectorStoreFactory;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.*;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.DigestInputStream;
import java.security.MessageDigest;
import java.util.*;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.locks.ReadWriteLock;
import java.util.concurrent.locks.ReentrantReadWriteLock;

public class KnowledgeBaseManager {

    private static final Logger log = LoggerFactory.getLogger(KnowledgeBaseManager.class);
    private static final String VDB_DIR = "./vdb";
    private static final String UPLOAD_DIR = "./upload_doc";

    private final Config cfg;
    private final MetaStore store;
    private final EmbeddingClient embClient;
    private final RerankClient rerankClient;
    private final ScheduledExecutorService scheduler;
    private final ReadWriteLock lock = new ReentrantReadWriteLock();
    private final Map<Long, VectorStore> stores = new HashMap<>();
    private volatile boolean running = true;

    public KnowledgeBaseManager(Config cfg, MetaStore store, EmbeddingClient embClient) {
        this.cfg = cfg;
        this.store = store;
        this.embClient = embClient;

        // Initialize rerank client if configured
        if (cfg.getApi().getRerankApiUri() != null && !cfg.getApi().getRerankApiUri().isEmpty()
                && cfg.getApi().getRerankModelName() != null && !cfg.getApi().getRerankModelName().isEmpty()) {
            this.rerankClient = new RerankClient(
                    cfg.getApi().getRerankApiUri(),
                    cfg.getApi().getRerankApiKey(),
                    cfg.getApi().getRerankModelName()
            );
        } else {
            this.rerankClient = null;
        }

        this.scheduler = Executors.newSingleThreadScheduledExecutor(r -> {
            Thread t = new Thread(r, "file-worker");
            t.setDaemon(true);
            return t;
        });
    }

    // ============================================================
    // Knowledge base CRUD
    // ============================================================

    public long createKB(String name, String uid, boolean isPublic) throws Exception {
        if (store.checkVdbNameExists(name, uid)) {
            throw new RuntimeException("知识库名称已存在: " + name);
        }

        long id = store.createVdb(name, uid, isPublic);
        VectorStore vs = getOrCreateStore(id);

        int dim = embClient.dimension();
        vs.ensureCollection(dim);

        return id;
    }

    public void deleteKB(long id, String uid) throws Exception {
        VdbInfo info = store.getVdbByID(id);
        if (info == null || !Objects.equals(info.getUid(), uid)) {
            throw new RuntimeException("无权删除该知识库");
        }

        lock.writeLock().lock();
        try {
            VectorStore vs = stores.remove(id);
            if (vs != null) {
                vs.purge();
                vs.close();
            }
        } finally {
            lock.writeLock().unlock();
        }

        List<VdbFileInfo> files = store.getFilesByVdbID(id);
        for (VdbFileInfo f : files) {
            new File(f.getFilePath()).delete();
        }

        store.deleteVdb(id);
    }

    public List<VdbInfo> getUserKBs(String uid) {
        return store.getUserVdbs(uid);
    }

    public List<VdbInfo> getPublicKBs(String uid) {
        return store.getPublicVdbs(uid);
    }

    public void setDefaultKB(long id, String uid) {
        store.setDefaultVdb(id, uid);
    }

    // ============================================================
    // File management
    // ============================================================

    public VdbFileInfo uploadFile(long vdbId, String uid, String fileName, InputStream fileStream) throws Exception {
        VdbInfo info = store.getVdbByID(vdbId);
        if (info == null || !Objects.equals(info.getUid(), uid)) {
            throw new RuntimeException("知识库不存在");
        }

        new File(UPLOAD_DIR).mkdirs();

        String taskId = String.valueOf(System.nanoTime());
        String savedName = taskId + "_" + fileName;
        String savedPath = UPLOAD_DIR + "/" + savedName;

        // Save file and compute MD5
        MessageDigest md5 = MessageDigest.getInstance("MD5");
        DigestInputStream dis = new DigestInputStream(fileStream, md5);

        try (FileOutputStream fos = new FileOutputStream(savedPath)) {
            dis.transferTo(fos);
        }
        String fileMd5 = bytesToHex(md5.digest());

        // Check for duplicate
        VdbFileInfo existing = store.checkFileMD5Exists(vdbId, fileMd5);
        if (existing != null) {
            deleteFile(existing.getId(), uid);
        }

        VdbFileInfo finfo = new VdbFileInfo();
        finfo.setName(fileName);
        finfo.setUid(uid);
        finfo.setVdbId(vdbId);
        finfo.setTaskId(taskId);
        finfo.setFilePath(savedPath);
        finfo.setPercent(0);
        finfo.setProcessInfo("文件已上传，等待处理");
        finfo.setFileMd5(fileMd5);

        long id = store.createFileInfo(finfo);
        finfo.setId(id);

        return finfo;
    }

    public List<VdbFileInfo> getFiles(long vdbId) {
        return store.getFilesByVdbID(vdbId);
    }

    public void deleteFile(long fileId, String uid) throws Exception {
        VdbFileInfo finfo = store.getFileByID(fileId);
        if (finfo == null || !Objects.equals(finfo.getUid(), uid)) {
            throw new RuntimeException("文件不存在");
        }

        String absPath = new File(finfo.getFilePath()).getCanonicalPath();
        deleteVectorsBySource(finfo.getVdbId(), absPath);

        new File(finfo.getFilePath()).delete();
        store.deleteFile(fileId);
    }

    // ============================================================
    // Search
    // ============================================================

    public String searchInKB(String query, long vdbId, String uid, int topK, double scoreThreshold) throws Exception {
        VectorStore vs = getOrCreateStore(vdbId);

        double[] queryVec = embClient.embedSingle(query);

        // Determine retrieval count: if rerank enabled, retrieve more first
        int retrieveN = topK;
        boolean useRerank = cfg.getKb().isRerankEnabled() && rerankClient != null;
        if (useRerank) {
            retrieveN = cfg.getKb().getRerankRetrieveN();
            if (retrieveN <= topK) {
                retrieveN = topK * 3;
            }
            if (retrieveN > 50) retrieveN = 50;
        }

        var results = vs.search(queryVec, retrieveN, scoreThreshold);

        // Rerank if enabled
        if (useRerank && results.size() > topK) {
            List<String> docs = results.stream()
                    .map(SearchResult::getContent)
                    .collect(java.util.stream.Collectors.toList());

            try {
                var rerankResults = rerankClient.rerank(query, docs, topK);
                // Reorder by rerank results
                List<SearchResult> reordered = new ArrayList<>();
                for (var rr : rerankResults) {
                    if (rr.getIndex() >= 0 && rr.getIndex() < results.size()) {
                        reordered.add(results.get(rr.getIndex()));
                    }
                }
                results = reordered;
            } catch (Exception e) {
                log.warn("rerank 失败，回退到原始排序", e);
                results = results.subList(0, Math.min(topK, results.size()));
            }
        }

        StringBuilder sb = new StringBuilder();
        for (var r : results) {
            String content = r.getContent().replace("\n", "");
            if (content.contains(".......................")) continue;
            sb.append(content).append("\n");
        }

        return sb.toString();
    }

    public String searchAllKBs(String query, String uid, int topK, double scoreThreshold) {
        List<VdbInfo> kbList = store.getUserVdbs(uid);
        if (kbList.isEmpty()) return "";

        StringBuilder allContext = new StringBuilder();
        for (VdbInfo kb : kbList) {
            try {
                String ctx = searchInKB(query, kb.getId(), uid, topK, scoreThreshold);
                if (ctx != null && !ctx.isEmpty()) {
                    allContext.append("[").append(kb.getName()).append("]\n");
                    allContext.append(ctx);
                }
            } catch (Exception e) {
                log.error("搜索知识库失败 kb={} error={}", kb.getName(), e.getMessage());
            }
        }

        return allContext.toString();
    }

    // ============================================================
    // File processing worker
    // ============================================================

    public void startFileWorker() {
        log.info("文件处理 worker 已启动");
        scheduler.scheduleWithFixedDelay(this::processPendingFiles, 0, 5, TimeUnit.SECONDS);
    }

    public void stopFileWorker() {
        running = false;
        scheduler.shutdown();
        log.info("文件处理 worker 已停止");
    }

    private void processPendingFiles() {
        if (!running) return;

        List<VdbFileInfo> files = store.getUnprocessedFiles();
        for (VdbFileInfo f : files) {
            if (!running) break;
            try {
                processFile(f);
            } catch (Exception e) {
                log.error("处理文件失败 file={} error={}", f.getName(), e.getMessage());
                store.updateFileProgress(f.getId(), 0, "处理失败: " + e.getMessage());
            }
        }
    }

    private void processFile(VdbFileInfo finfo) throws Exception {
        log.info("开始处理文件 name={} id={}", finfo.getName(), finfo.getId());
        store.updateFileProgress(finfo.getId(), 1, "开始处理文档");

        // Extract text
        String ext = finfo.getFilePath().toLowerCase();
        String text;
        if (ext.endsWith(".pdf") || ext.endsWith(".docx") || ext.endsWith(".xlsx") || ext.endsWith(".xls")) {
            text = TextExtractor.extractAndSaveText(finfo.getFilePath());
        } else {
            text = Files.readString(Path.of(finfo.getFilePath()));
        }

        if (text == null || text.trim().isEmpty()) {
            store.updateFileProgress(finfo.getId(), 100, "文件内容为空");
            return;
        }

        // Split text
        List<String> chunks = splitText(text, cfg.getKb().getChunkSize(), cfg.getKb().getChunkOverlap());
        if (chunks.isEmpty()) {
            store.updateFileProgress(finfo.getId(), 100, "无可切分的文本内容");
            return;
        }

        log.info("文件已切分 name={} chunks={}", finfo.getName(), chunks.size());
        store.updateFileProgress(finfo.getId(), 5,
                String.format("已切分为 %d 个文本块，开始向量化", chunks.size()));

        // Initialize vector store
        VectorStore vs = getOrCreateStore(finfo.getVdbId());
        int dim = embClient.dimension();
        vs.ensureCollection(dim);

        // Batch vectorization
        int batchSize = 10;
        String fileName = new File(finfo.getFilePath()).getName();
        String absPath = new File(finfo.getFilePath()).getCanonicalPath();
        int totalChunks = chunks.size();

        for (int i = 0; i < totalChunks; i += batchSize) {
            if (!running) break;

            int end = Math.min(i + batchSize, totalChunks);
            List<String> batch = chunks.subList(i, end);

            // Batch embedding
            List<double[]> embeddings = embClient.embed(batch);

            // Build records
            List<VectorRecord> records = new ArrayList<>();
            for (int j = 0; j < batch.size(); j++) {
                VectorRecord rec = new VectorRecord();
                rec.setId(fileName + "_chunk_" + (i + j));
                rec.setVector(embeddings.get(j));
                rec.setContent(batch.get(j));
                rec.setMeta(Map.of("source", absPath));
                records.add(rec);
            }

            // Insert
            vs.insert(records);

            // Update progress
            double percent = (double) end / totalChunks * 100;
            if (percent > 99) percent = 99;
            store.updateFileProgress(finfo.getId(), percent,
                    String.format("已处理 %d/%d 个文本块", end, totalChunks));
        }

        store.updateFileProgress(finfo.getId(), 100,
                String.format("处理完成，共 %d 个文本块", totalChunks));
        log.info("文件处理完成 name={}", finfo.getName());
    }

    // ============================================================
    // Internal methods
    // ============================================================

    private VectorStore getOrCreateStore(long vdbId) throws Exception {
        lock.readLock().lock();
        try {
            VectorStore vs = stores.get(vdbId);
            if (vs != null) return vs;
        } finally {
            lock.readLock().unlock();
        }

        lock.writeLock().lock();
        try {
            VectorStore vs = stores.get(vdbId);
            if (vs != null) return vs;

            new File(VDB_DIR).mkdirs();
            vs = VectorStoreFactory.create(cfg, vdbId);
            stores.put(vdbId, vs);
            return vs;
        } finally {
            lock.writeLock().unlock();
        }
    }

    private void deleteVectorsBySource(long vdbId, String source) {
        lock.readLock().lock();
        try {
            VectorStore vs = stores.get(vdbId);
            if (vs != null) {
                vs.deleteBySource(source);
            }
        } catch (Exception e) {
            log.error("删除向量失败", e);
        } finally {
            lock.readLock().unlock();
        }
    }

    // ============================================================
    // Text splitting
    // ============================================================

    static List<String> splitText(String text, int chunkSize, int chunkOverlap) {
        if (chunkSize <= 0) chunkSize = 300;

        String[] paragraphs = text.split("\n");
        List<String> chunks = new ArrayList<>();
        String current = "";

        for (String para : paragraphs) {
            para = para.trim();
            if (para.isEmpty()) continue;

            int len = para.codePointCount(0, para.length());
            if (len <= chunkSize) {
                if (current.isEmpty()) {
                    current = para;
                } else {
                    String combined = current + "\n" + para;
                    int combinedLen = combined.codePointCount(0, combined.length());
                    if (combinedLen <= chunkSize) {
                        current = combined;
                    } else {
                        chunks.add(current);
                        current = para;
                    }
                }
            } else {
                if (!current.isEmpty()) {
                    chunks.add(current);
                    current = "";
                }
                int step = chunkSize - chunkOverlap;
                int pos = 0;
                while (pos < para.length()) {
                    int end = Math.min(pos + chunkSize, para.length());
                    chunks.add(para.substring(pos, end));
                    if (end == para.length()) break;
                    pos += step;
                }
            }
        }

        if (!current.isEmpty()) {
            chunks.add(current);
        }

        return chunks;
    }

    private static String bytesToHex(byte[] bytes) {
        StringBuilder sb = new StringBuilder();
        for (byte b : bytes) {
            sb.append(String.format("%02x", b));
        }
        return sb.toString();
    }
}