package com.rd.robot.server;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.rd.robot.kb.KnowledgeBaseManager;
import com.rd.robot.model.Config;
import com.rd.robot.model.VdbFileInfo;
import com.rd.robot.model.VdbInfo;
import com.rd.robot.store.SQLiteStore;

import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.handler.codec.http.HttpHeaderNames;
import io.netty.util.CharsetUtil;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.util.*;

public class VdbHandler {

    private static final Logger log = LoggerFactory.getLogger(VdbHandler.class);
    private final ObjectMapper mapper = new ObjectMapper();

    private static final Set<String> ALLOWED_EXTS = Set.of(".txt", ".md", ".pdf", ".docx", ".xlsx");

    private final Config cfg;
    private final KnowledgeBaseManager kbMgr;
    private final SQLiteStore store;

    public VdbHandler(Config cfg, KnowledgeBaseManager kbMgr, SQLiteStore store) {
        this.cfg = cfg;
        this.kbMgr = kbMgr;
        this.store = store;
    }

    // ============================================================
    // POST /vdb/my/list
    // ============================================================
    public void myList(ChannelHandlerContext ctx, FullHttpRequest req) {
        String uid = getUid(req);
        try {
            List<VdbInfo> list = kbMgr.getUserKBs(uid);
            if (list == null) list = List.of();
            sendJson(ctx, 200, Map.of("data", list));
        } catch (Exception e) {
            sendError(ctx, e.getMessage());
        }
    }

    // ============================================================
    // POST /vdb/pub/list
    // ============================================================
    public void pubList(ChannelHandlerContext ctx, FullHttpRequest req) {
        String uid = getUid(req);
        try {
            List<VdbInfo> list = kbMgr.getPublicKBs(uid);
            if (list == null) list = List.of();
            sendJson(ctx, 200, Map.of("data", list));
        } catch (Exception e) {
            sendError(ctx, e.getMessage());
        }
    }

    // ============================================================
    // POST /vdb/file/list
    // ============================================================
    public void fileList(ChannelHandlerContext ctx, FullHttpRequest req) {
        long vdbId = getLongParam(req, "vdb_id");
        try {
            List<VdbFileInfo> files = kbMgr.getFiles(vdbId);
            if (files == null) files = List.of();
            sendJson(ctx, 200, Map.of("data", files));
        } catch (Exception e) {
            sendError(ctx, e.getMessage());
        }
    }

    // ============================================================
    // POST /vdb/set/default
    // ============================================================
    public void setDefault(ChannelHandlerContext ctx, FullHttpRequest req) {
        String uid = getUid(req);
        long id = getLongParam(req, "id");
        try {
            kbMgr.setDefaultKB(id, uid);
            sendJson(ctx, 200, Map.of("status", "ok"));
        } catch (Exception e) {
            sendError(ctx, e.getMessage());
        }
    }

    // ============================================================
    // POST /vdb/create
    // ============================================================
    public void create(ChannelHandlerContext ctx, FullHttpRequest req) {
        String uid = getUid(req);
        String name = HttpServer.getParam(req, "name");
        String isPublicStr = HttpServer.getParam(req, "is_public");
        boolean isPublic = "true".equals(isPublicStr) || "1".equals(isPublicStr);

        if (name == null || name.isEmpty()) {
            sendJson(ctx, 400, Map.of("error", "知识库名称不能为空"));
            return;
        }

        try {
            long id = kbMgr.createKB(name, uid, isPublic);
            sendJson(ctx, 200, Map.of("status", "ok", "id", id));
        } catch (Exception e) {
            sendError(ctx, e.getMessage());
        }
    }

    // ============================================================
    // POST /vdb/delete
    // ============================================================
    public void delete(ChannelHandlerContext ctx, FullHttpRequest req) {
        String uid = getUid(req);
        long id = getLongParam(req, "id");
        try {
            kbMgr.deleteKB(id, uid);
            sendJson(ctx, 200, Map.of("status", "ok"));
        } catch (Exception e) {
            sendError(ctx, e.getMessage());
        }
    }

    // ============================================================
    // POST /vdb/upload  (multipart/form-data)
    // ============================================================
    public void upload(ChannelHandlerContext ctx, FullHttpRequest req) {
        String contentType = req.headers().get(HttpHeaderNames.CONTENT_TYPE);
        if (contentType == null || !contentType.contains("multipart/form-data")) {
            sendJson(ctx, 400, Map.of("error", "请使用 multipart/form-data 上传文件"));
            return;
        }

        try {
            MultipartForm form = parseMultipart(req, contentType);
            String uid = form.getField("uid");
            if (uid == null || uid.isEmpty()) uid = "default";

            String vdbIdStr = form.getField("vdb_id");
            if (vdbIdStr == null) {
                sendJson(ctx, 400, Map.of("error", "无效的知识库 ID"));
                return;
            }
            long vdbId = Long.parseLong(vdbIdStr);

            MultipartForm.FilePart file = form.getFile("file");
            if (file == null) {
                sendJson(ctx, 400, Map.of("error", "请选择文件"));
                return;
            }

            // 检查文件类型
            String fileName = file.filename;
            String ext = fileName.toLowerCase();
            int dotIdx = ext.lastIndexOf('.');
            if (dotIdx < 0 || !ALLOWED_EXTS.contains(ext.substring(dotIdx))) {
                sendJson(ctx, 400, Map.of("error", "不支持的文件格式，支持: txt, md, pdf, docx, xlsx"));
                return;
            }

            InputStream fileStream = new ByteArrayInputStream(file.content);
            VdbFileInfo finfo = kbMgr.uploadFile(vdbId, uid, fileName, fileStream);

            sendJson(ctx, 200, Map.of("status", "ok", "file", finfo));
        } catch (Exception e) {
            log.error("上传文件失败", e);
            sendError(ctx, e.getMessage());
        }
    }

    // ============================================================
    // POST /vdb/process/info
    // ============================================================
    public void processInfo(ChannelHandlerContext ctx, FullHttpRequest req) {
        long fileId = getLongParam(req, "file_id");
        try {
            VdbFileInfo finfo = store.getFileByID(fileId);
            sendJson(ctx, 200, Map.of("data", finfo != null ? finfo : Map.of()));
        } catch (Exception e) {
            sendError(ctx, e.getMessage());
        }
    }

    // ============================================================
    // POST /vdb/search
    // ============================================================
    public void search(ChannelHandlerContext ctx, FullHttpRequest req) {
        String uid = getUid(req);
        long vdbId = getLongParam(req, "vdb_id");
        String query = HttpServer.getParam(req, "query");

        if (query == null || query.isEmpty()) {
            sendJson(ctx, 400, Map.of("error", "请输入搜索内容"));
            return;
        }

        try {
            String result = kbMgr.searchInKB(query, vdbId, uid, 5, 0.1);
            sendJson(ctx, 200, Map.of("data", result));
        } catch (Exception e) {
            sendError(ctx, e.getMessage());
        }
    }

    // ============================================================
    // POST /vdb/file/delete
    // ============================================================
    public void fileDelete(ChannelHandlerContext ctx, FullHttpRequest req) {
        String uid = getUid(req);
        long fileId = getLongParam(req, "file_id");
        try {
            kbMgr.deleteFile(fileId, uid);
            sendJson(ctx, 200, Map.of("status", "ok"));
        } catch (Exception e) {
            sendError(ctx, e.getMessage());
        }
    }

    // ============================================================
    // 辅助方法
    // ============================================================

    private String getUid(FullHttpRequest req) {
        String uid = HttpServer.getParam(req, "uid");
        return uid != null && !uid.isEmpty() ? uid : "default";
    }

    private long getLongParam(FullHttpRequest req, String key) {
        String val = HttpServer.getParam(req, key);
        if (val == null || val.isEmpty()) return 0;
        try {
            return Long.parseLong(val);
        } catch (NumberFormatException e) {
            return 0;
        }
    }

    private void sendJson(ChannelHandlerContext ctx, int statusCode, Object data) {
        try {
            String json = mapper.writeValueAsString(data);
            HttpServer.sendJson(ctx, statusCode, json);
        } catch (Exception e) {
            HttpServer.sendError(ctx, io.netty.handler.codec.http.HttpResponseStatus.INTERNAL_SERVER_ERROR,
                    "JSON 序列化失败");
        }
    }

    private void sendError(ChannelHandlerContext ctx, String message) {
        sendJson(ctx, 500, Map.of("error", message));
    }

    // ============================================================
    // Multipart 解析
    // ============================================================

    private MultipartForm parseMultipart(FullHttpRequest req, String contentType) {
        // 提取 boundary
        int boundaryIdx = contentType.indexOf("boundary=");
        if (boundaryIdx < 0) {
            throw new RuntimeException("缺少 boundary");
        }
        String boundary = "--" + contentType.substring(boundaryIdx + 9).trim();

        byte[] body = new byte[req.content().readableBytes()];
        req.content().readBytes(body);

        MultipartForm form = new MultipartForm();
        int pos = 0;
        byte[] boundaryBytes = boundary.getBytes(CharsetUtil.UTF_8);

        while (pos < body.length) {
            // 找下一个 boundary
            int nextBoundary = indexOf(body, boundaryBytes, pos);
            if (nextBoundary < 0) break;

            int headerStart = nextBoundary + boundaryBytes.length;
            if (headerStart >= body.length) break;

            // 跳过 \r\n
            if (body[headerStart] == '\r' && headerStart + 1 < body.length && body[headerStart + 1] == '\n') {
                headerStart += 2;
            }

            // 找 headers 结束位置 (\r\n\r\n)
            int headerEnd = indexOf(body, new byte[]{'\r', '\n', '\r', '\n'}, headerStart);
            if (headerEnd < 0) break;

            String headers = new String(body, headerStart, headerEnd - headerStart, CharsetUtil.UTF_8);

            // 找内容结束位置（下一个 boundary 或 boundary--）
            int contentStart = headerEnd + 4;
            int contentEnd = indexOf(body, boundaryBytes, contentStart);
            if (contentEnd < 0) break;

            // 内容去除结尾 \r\n
            int contentActualEnd = contentEnd;
            if (contentActualEnd > contentStart && body[contentActualEnd - 1] == '\n') contentActualEnd--;
            if (contentActualEnd > contentStart && body[contentActualEnd - 1] == '\r') contentActualEnd--;

            byte[] content = Arrays.copyOfRange(body, contentStart, contentActualEnd);

            // 解析 headers
            String lowerHeaders = headers.toLowerCase();
            if (lowerHeaders.contains("filename=")) {
                // 文件部分
                String filename = extractHeaderValue(headers, "filename=\"", "\"");
                MultipartForm.FilePart file = new MultipartForm.FilePart();
                file.fieldName = extractHeaderValue(headers, "name=\"", "\"");
                file.filename = filename;
                file.contentType = extractHeaderValue(headers, "Content-Type: ", "\r\n");
                file.content = content;
                form.files.put(file.fieldName, file);
            } else {
                // 普通字段
                String fieldName = extractHeaderValue(headers, "name=\"", "\"");
                String value = new String(content, CharsetUtil.UTF_8);
                form.fields.put(fieldName, value);
            }

            pos = contentEnd;
        }

        return form;
    }

    private int indexOf(byte[] data, byte[] pattern, int from) {
        for (int i = from; i <= data.length - pattern.length; i++) {
            boolean match = true;
            for (int j = 0; j < pattern.length; j++) {
                if (data[i + j] != pattern[j]) {
                    match = false;
                    break;
                }
            }
            if (match) return i;
        }
        return -1;
    }

    private String extractHeaderValue(String headers, String prefix, String suffix) {
        int idx = headers.indexOf(prefix);
        if (idx < 0) return "";
        idx += prefix.length();
        int endIdx = headers.indexOf(suffix, idx);
        if (endIdx < 0) return headers.substring(idx);
        return headers.substring(idx, endIdx);
    }

    // Multipart 表单数据结构
    private static class MultipartForm {
        Map<String, String> fields = new HashMap<>();
        Map<String, FilePart> files = new HashMap<>();

        String getField(String name) {
            return fields.get(name);
        }

        FilePart getFile(String name) {
            return files.get(name);
        }

        static class FilePart {
            String fieldName;
            String filename;
            String contentType;
            byte[] content;
        }
    }
}
