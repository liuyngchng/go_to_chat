package com.rd.robot.web.controller;

import com.fasterxml.jackson.databind.ObjectMapper;
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
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Authentication controller — login, logout, session management.
 */
public class AuthController {

    private static final Logger log = LoggerFactory.getLogger(AuthController.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();

    private final MetaStore metaStore;
    private final ConcurrentHashMap<String, Instant> onlineAgents = new ConcurrentHashMap<>();

    public AuthController(MetaStore metaStore) {
        this.metaStore = metaStore;
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

            String token = TokenProvider.generateToken(user.getUserName(), user.getRole());

            // Track online agents
            if (user.getRole() == User.ROLE_AGENT) {
                onlineAgents.put(user.getUserName(), Instant.now());
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
                onlineAgents.remove(user.getUserName());
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
        StringBuilder sb = new StringBuilder("{\"agents\":[");
        boolean first = true;
        for (var entry : onlineAgents.entrySet()) {
            if (!first) sb.append(",");
            first = false;
            sb.append("{");
            sb.append("\"user_name\":\"").append(entry.getKey()).append("\"");
            sb.append(",\"login_time\":\"").append(entry.getValue().toString()).append("\"");
            // Get note from DB
            try {
                User user = metaStore.getUserByName(entry.getKey());
                if (user != null && user.getNote() != null) {
                    sb.append(",\"note\":\"").append(escapeJson(user.getNote())).append("\"");
                }
            } catch (Exception ignored) {}
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