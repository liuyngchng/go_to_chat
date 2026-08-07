package com.rd.robot.web.controller;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.rd.robot.model.Config;
import com.rd.robot.model.LoginRequest;
import com.rd.robot.model.User;
import com.rd.robot.repository.MetaStore;
import com.rd.robot.security.TokenProvider;
import com.rd.robot.web.server.HttpServer;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.handler.codec.http.HttpHeaderNames;
import io.netty.util.CharsetUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Instant;
import java.util.List;
import java.util.Map;

/**
 * Authentication controller — login, logout, session management.
 */
public class AuthController {

    private static final Logger log = LoggerFactory.getLogger(AuthController.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();

    private final Config cfg;
    private final MetaStore metaStore;
    private final PresenceStore presenceStore;

    public AuthController(Config cfg, MetaStore metaStore, PresenceStore presenceStore) {
        this.cfg = cfg;
        this.metaStore = metaStore;
        this.presenceStore = presenceStore;
    }

    /**
     * POST /api/login — JSON login
     */
    public void login(ChannelHandlerContext ctx, FullHttpRequest request) {
        try {
            String body = request.content().toString(CharsetUtil.UTF_8);
            LoginRequest req = MAPPER.readValue(body, LoginRequest.class);

            if (req.getUserName() == null || req.getUserName().isEmpty()
                    || req.getUserPwd() == null || req.getUserPwd().isEmpty()) {
                HttpServer.sendJson(ctx, 400, "{\"error\":\"用户名和密码不能为空\"}");
                return;
            }

            String md5Pwd = TokenProvider.md5Hash(req.getUserPwd());
            User user = metaStore.getUserByLogin(req.getUserName(), md5Pwd);

            if (user == null) {
                HttpServer.sendJson(ctx, 401, "{\"error\":\"用户名或密码错误\"}");
                return;
            }

            // admin 实例：仅管理员可登录
            if (cfg.getServer().isAdminOnly() && user.getRole() != User.ROLE_ADMIN) {
                HttpServer.sendJson(ctx, 403, "{\"error\":\"此账号无法访问管理后台\"}");
                return;
            }

            String token = TokenProvider.generateToken(user.getUserName(), user.getRole());

            // Track online agents
            if (user.getRole() == User.ROLE_AGENT) {
                presenceStore.setPresence(user.getUserName(), System.currentTimeMillis());
            }

            Map<String, Object> resp = Map.of(
                    "status", "ok",
                    "token", token,
                    "user_name", user.getUserName(),
                    "role", user.getRole()
            );
            HttpServer.sendJson(ctx, 200, MAPPER.writeValueAsString(resp));

        } catch (Exception e) {
            log.error("登录失败", e);
            HttpServer.sendJson(ctx, 500, "{\"error\":\"登录失败: " + e.getMessage() + "\"}");
        }
    }

    /**
     * POST /api/logout — logout
     */
    public void logout(ChannelHandlerContext ctx, FullHttpRequest request) {
        String token = extractToken(request);
        if (token != null) {
            User user = TokenProvider.parseToken(token);
            if (user != null) {
                presenceStore.removePresence(user.getUserName());
            }
        }
        HttpServer.sendJson(ctx, 200, "{\"status\":\"ok\"}");
    }

    /**
     * GET /api/me — current user info
     */
    public void me(ChannelHandlerContext ctx, FullHttpRequest request) {
        String token = extractToken(request);
        if (token == null) {
            HttpServer.sendJson(ctx, 401, "{\"error\":\"未登录\"}");
            return;
        }

        User user = TokenProvider.parseToken(token);
        if (user == null) {
            HttpServer.sendJson(ctx, 401, "{\"error\":\"token 无效或已过期\"}");
            return;
        }

        HttpServer.sendJson(ctx, 200, String.format(
                "{\"user_name\":\"%s\",\"role\":%d}", user.getUserName(), user.getRole()));
    }

    /**
     * GET /api/agents — online agents list
     */
    public void getOnlineAgents(ChannelHandlerContext ctx, FullHttpRequest request) {
        List<Map<String, Object>> agents = presenceStore.getOnlineAgents();

        StringBuilder sb = new StringBuilder("{\"agents\":[");
        boolean first = true;
        for (var info : agents) {
            if (!first) sb.append(",");
            first = false;
            sb.append("{");
            boolean firstField = true;
            for (var entry : info.entrySet()) {
                if (!firstField) sb.append(",");
                firstField = false;
                sb.append("\"").append(entry.getKey()).append("\":\"")
                        .append(escapeJson(String.valueOf(entry.getValue()))).append("\"");
            }
            sb.append("}");
        }
        sb.append("]}");
        HttpServer.sendJson(ctx, 200, sb.toString());
    }

    // ============================================================
    // Token extraction helpers
    // ============================================================

    public static String extractToken(FullHttpRequest request) {
        String token = HttpServer.getQueryParam(request, "t");
        if (token != null) return token;
        String auth = request.headers().get(HttpHeaderNames.AUTHORIZATION);
        if (auth != null && auth.startsWith("Bearer ")) {
            return auth.substring(7);
        }
        return null;
    }

    public static User parseUserFromRequest(FullHttpRequest request) {
        String token = extractToken(request);
        if (token == null) return null;
        return TokenProvider.parseToken(token);
    }

    private static String escapeJson(String s) {
        if (s == null) return "";
        StringBuilder sb = new StringBuilder();
        for (char c : s.toCharArray()) {
            switch (c) {
                case '"': sb.append("\\\""); break;
                case '\\': sb.append("\\\\"); break;
                case '\n': sb.append("\\n"); break;
                case '\r': sb.append("\\r"); break;
                case '\t': sb.append("\\t"); break;
                default: sb.append(c);
            }
        }
        return sb.toString();
    }
}