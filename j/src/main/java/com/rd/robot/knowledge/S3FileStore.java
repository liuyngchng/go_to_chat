package com.rd.robot.knowledge;

import com.rd.robot.model.Config;
import io.minio.*;
import io.minio.errors.*;

import java.io.*;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.InvalidKeyException;
import java.security.NoSuchAlgorithmException;

/**
 * S3/MinIO 对象存储实现（集群模式）。
 */
public class S3FileStore implements FileStore {

    private final MinioClient client;
    private final String bucket;

    public S3FileStore(Config cfg) {
        this.bucket = cfg.getOss().getBucket();
        this.client = MinioClient.builder()
                .endpoint(cfg.getOss().getEndpoint())
                .credentials(cfg.getOss().getAccessKey(), cfg.getOss().getSecretKey())
                .build();

        // 确保 bucket 存在
        try {
            boolean found = client.bucketExists(BucketExistsArgs.builder().bucket(bucket).build());
            if (!found) {
                client.makeBucket(MakeBucketArgs.builder().bucket(bucket).build());
            }
        } catch (Exception e) {
            throw new RuntimeException("初始化 S3 bucket 失败: " + bucket, e);
        }
    }

    @Override
    public void mkdirAll(String dir) throws Exception {
        // S3 不需要显式创建目录
    }

    @Override
    public long save(String path, InputStream in) throws Exception {
        // 对 S3 路径做规范化：去掉 ./ 前缀
        String key = path.replaceFirst("^\\./", "");
        // 先读取全部内容以获取长度
        byte[] data = in.readAllBytes();
        try (ByteArrayInputStream bis = new ByteArrayInputStream(data)) {
            client.putObject(PutObjectArgs.builder()
                    .bucket(bucket)
                    .object(key)
                    .stream(bis, data.length, -1)
                    .build());
        }
        return data.length;
    }

    @Override
    public byte[] readAll(String path) throws Exception {
        String key = path.replaceFirst("^\\./", "");
        try (InputStream in = client.getObject(GetObjectArgs.builder()
                .bucket(bucket).object(key).build())) {
            return in.readAllBytes();
        }
    }

    @Override
    public void delete(String path) throws Exception {
        String key = path.replaceFirst("^\\./", "");
        client.removeObject(RemoveObjectArgs.builder().bucket(bucket).object(key).build());
    }

    @Override
    public boolean exists(String path) {
        String key = path.replaceFirst("^\\./", "");
        try {
            client.statObject(StatObjectArgs.builder().bucket(bucket).object(key).build());
            return true;
        } catch (Exception e) {
            return false;
        }
    }

    /** 将 S3 文件下载到本地临时文件（用于文档解析），返回 (临时路径, 清理回调) */
    public String downloadToTemp(String remotePath) throws Exception {
        String key = remotePath.replaceFirst("^\\./", "");
        Path tmp = Files.createTempFile("kb_doc_", ".tmp");
        try (InputStream in = client.getObject(GetObjectArgs.builder()
                .bucket(bucket).object(key).build())) {
            Files.write(tmp, in.readAllBytes());
        }
        return tmp.toString();
    }

    /** 删除临时文件 */
    public void cleanTemp(String tmpPath) {
        try {
            Files.deleteIfExists(Path.of(tmpPath));
        } catch (IOException ignored) {}
    }
}