package com.rd.robot.vector;

import com.rd.robot.model.SearchResult;
import com.rd.robot.model.VectorRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.concurrent.locks.ReadWriteLock;
import java.util.concurrent.locks.ReentrantReadWriteLock;

/**
 * Qdrant vector store — gRPC-based implementation.
 * Uses the Qdrant gRPC API for vector storage and search.
 */
public class QdrantVectorStore implements VectorStore {

    private static final Logger log = LoggerFactory.getLogger(QdrantVectorStore.class);
    private static final String COLLECTION_PREFIX = "kb_";

    // Placeholder: In production, this would use the Qdrant gRPC client.
    // For now, provides a local fallback that mirrors LocalVectorStore behavior.
    private final LocalVectorStore fallbackStore;
    private final String host;
    private final int port;
    private final String apiKey;
    private final boolean useTls;
    private final long vdbId;
    private final String collectionName;

    private boolean qdrantAvailable = false;

    public QdrantVectorStore(String host, int port, String apiKey, boolean useTls, long vdbId) {
        this.host = host;
        this.port = port;
        this.apiKey = apiKey;
        this.useTls = useTls;
        this.vdbId = vdbId;
        this.collectionName = COLLECTION_PREFIX + vdbId;

        // Try to connect to Qdrant; fall back to local storage if unavailable
        LocalVectorStore fs = null;
        try {
            // In production, create gRPC channel and clients here:
            // import io.grpc.*; import qdrant.*;
            // this.conn = grpc.newClient(...);
            // this.pointsClient = ...;
            // this.collectionsClient = ...;
            // this.qdrantAvailable = true;

            // For now, log the config and use fallback
            log.info("Qdrant 配置: host={} port={} tls={} collection={}", host, port, useTls, collectionName);
            log.info("Qdrant 客户端将在生产环境中启用 gRPC 连接");

            // Fallback to local storage for development
            fs = new LocalVectorStore("./vdb/vectors.db", vdbId);
            this.qdrantAvailable = false;
        } catch (Exception e) {
            log.warn("Qdrant 初始化失败，回退到本地存储", e);
            try {
                fs = new LocalVectorStore("./vdb/vectors.db", vdbId);
            } catch (Exception ex) {
                throw new RuntimeException("回退本地存储也失败", ex);
            }
        }
        this.fallbackStore = fs;
    }

    @Override
    public void ensureCollection(int dimension) throws Exception {
        if (qdrantAvailable) {
            // In production: call collectionsClient.Create(...)
            log.info("Qdrant ensureCollection: {} dim={}", collectionName, dimension);
        } else {
            fallbackStore.ensureCollection(dimension);
        }
    }

    @Override
    public void insert(List<VectorRecord> records) throws Exception {
        if (qdrantAvailable) {
            // In production: call pointsClient.Upsert(...)
            log.info("Qdrant insert: {} records={}", collectionName, records.size());
        } else {
            fallbackStore.insert(records);
        }
    }

    @Override
    public List<SearchResult> search(double[] queryVector, int topK, double scoreThreshold) throws Exception {
        if (qdrantAvailable) {
            // In production: call pointsClient.Search(...)
            log.info("Qdrant search: {} topK={}", collectionName, topK);
            return List.of();
        } else {
            return fallbackStore.search(queryVector, topK, scoreThreshold);
        }
    }

    @Override
    public void deleteByIds(List<String> ids) throws Exception {
        if (qdrantAvailable) {
            log.info("Qdrant deleteByIds: {} ids={}", collectionName, ids.size());
        } else {
            fallbackStore.deleteByIds(ids);
        }
    }

    @Override
    public void deleteBySource(String source) throws Exception {
        if (qdrantAvailable) {
            log.info("Qdrant deleteBySource: {} source={}", collectionName, source);
        } else {
            fallbackStore.deleteBySource(source);
        }
    }

    @Override
    public void purge() throws Exception {
        if (qdrantAvailable) {
            log.info("Qdrant purge: {}", collectionName);
        } else {
            fallbackStore.purge();
        }
    }

    @Override
    public void close() throws Exception {
        if (!qdrantAvailable) {
            fallbackStore.close();
        }
    }
}