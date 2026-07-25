package com.rd.robot.server;

import com.rd.robot.model.Config;

import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;

import java.io.InputStream;
import java.nio.charset.StandardCharsets;

public class PageHandler {

    private final Config cfg;

    public PageHandler(Config cfg) {
        this.cfg = cfg;
    }

    // ============================================================
    // GET / — 聊天主页面
    // ============================================================

    public void index(ChannelHandlerContext ctx, FullHttpRequest request) throws Exception {
        String uid = getUid(request);
        String html = loadTemplate("templates/index.html")
                .replace("{{sys_name}}", cfg.getSys().getName())
                .replace("{{uid}}", uid)
                .replace("{{app_source}}", "csm");
        HttpServer.sendHtml(ctx, html);
    }

    // ============================================================
    // GET /vdb/idx — 知识库管理页面
    // ============================================================

    public void vdbIndex(ChannelHandlerContext ctx, FullHttpRequest request) throws Exception {
        String uid = getUid(request);
        String html = loadTemplate("templates/vdb.html")
                .replace("{{sys_name}}", cfg.getSys().getName())
                .replace("{{uid}}", uid)
                .replace("{{app_source}}", "csm");
        HttpServer.sendHtml(ctx, html);
    }

    // ============================================================
    // 辅助方法
    // ============================================================

    private String loadTemplate(String path) throws Exception {
        try (InputStream in = getClass().getClassLoader().getResourceAsStream(path)) {
            if (in == null) {
                throw new RuntimeException("模板文件不存在: " + path);
            }
            return new String(in.readAllBytes(), StandardCharsets.UTF_8);
        }
    }

    private String getUid(FullHttpRequest request) {
        String uid = HttpServer.getQueryParam(request, "uid");
        return uid != null ? uid : "default";
    }
}
