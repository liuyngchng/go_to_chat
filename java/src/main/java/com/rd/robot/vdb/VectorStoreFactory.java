package com.rd.robot.vdb;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.sql.SQLException;

public class VectorStoreFactory {

    private static final Logger log = LoggerFactory.getLogger(VectorStoreFactory.class);

    public static VectorStore create(String milvusUri, String milvusToken, String storePath, long vdbId) {
        if (milvusUri != null && !milvusUri.isEmpty()
                && (milvusUri.startsWith("http://") || milvusUri.startsWith("https://"))) {
            log.info("使用远程 Milvus uri={}", milvusUri);
            return new MilvusVectorStore(milvusUri, milvusToken);
        }

        log.info("使用本地向量存储 path={}", storePath);
        try {
            return new LocalVectorStore(storePath, vdbId);
        } catch (SQLException e) {
            throw new RuntimeException("创建本地向量存储失败", e);
        }
    }
}
