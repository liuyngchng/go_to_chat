package com.rd.robot.knowledge;

import java.io.InputStream;

/**
 * 文件存储抽象，支持本地文件系统和 S3/MinIO 两种实现。
 */
public interface FileStore {

    /** 确保目录存在 */
    void mkdirAll(String dir) throws Exception;

    /** 保存文件（从流），返回写入字节数 */
    long save(String path, InputStream in) throws Exception;

    /** 读取全部内容 */
    byte[] readAll(String path) throws Exception;

    /** 删除文件 */
    void delete(String path) throws Exception;

    /** 文件是否存在 */
    boolean exists(String path);
}