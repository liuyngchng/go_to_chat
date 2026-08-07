package com.rd.robot.knowledge;

import java.io.*;
import java.nio.file.Files;
import java.nio.file.Path;

/**
 * 本地文件系统实现（单例模式）。
 */
public class LocalFileStore implements FileStore {

    @Override
    public void mkdirAll(String dir) throws Exception {
        new File(dir).mkdirs();
    }

    @Override
    public long save(String path, InputStream in) throws Exception {
        File f = new File(path);
        // 确保父目录存在
        File parent = f.getParentFile();
        if (parent != null && !parent.exists()) {
            parent.mkdirs();
        }
        long total = 0;
        try (FileOutputStream fos = new FileOutputStream(f)) {
            byte[] buf = new byte[8192];
            int n;
            while ((n = in.read(buf)) != -1) {
                fos.write(buf, 0, n);
                total += n;
            }
        }
        return total;
    }

    @Override
    public byte[] readAll(String path) throws Exception {
        return Files.readAllBytes(Path.of(path));
    }

    @Override
    public void delete(String path) throws Exception {
        new File(path).delete();
    }

    @Override
    public boolean exists(String path) {
        return new File(path).exists();
    }
}