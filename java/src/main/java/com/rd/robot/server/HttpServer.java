package com.rd.robot.server;

import io.netty.bootstrap.ServerBootstrap;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.Unpooled;
import io.netty.channel.*;
import io.netty.channel.nio.NioEventLoopGroup;
import io.netty.channel.socket.nio.NioServerSocketChannel;
import io.netty.handler.codec.http.*;
import io.netty.handler.stream.ChunkedWriteHandler;
import io.netty.util.CharsetUtil;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.InputStream;

public class HttpServer {

    private static final Logger log = LoggerFactory.getLogger(HttpServer.class);

    private final int port;
    private final Router router;
    private final EventLoopGroup bossGroup;
    private final EventLoopGroup workerGroup;
    private Channel channel;

    public HttpServer(int port, Router router) {
        this.port = port;
        this.router = router;
        this.bossGroup = new NioEventLoopGroup(1);
        this.workerGroup = new NioEventLoopGroup();
    }

    public void start() {
        try {
            ServerBootstrap bootstrap = new ServerBootstrap();
            bootstrap.group(bossGroup, workerGroup)
                    .channel(NioServerSocketChannel.class)
                    .childHandler(new ChannelInitializer<Channel>() {
                        @Override
                        protected void initChannel(Channel ch) {
                            ChannelPipeline p = ch.pipeline();
                            p.addLast(new HttpServerCodec());
                            p.addLast(new HttpObjectAggregator(64 * 1024 * 1024)); // 64MB
                            p.addLast(new ChunkedWriteHandler());
                            p.addLast(new ServerHandler(router));
                        }
                    });

            channel = bootstrap.bind(port).sync().channel();
        } catch (Exception e) {
            throw new RuntimeException("启动 HTTP 服务器失败", e);
        }
    }

    public void stop() {
        try {
            if (channel != null) {
                channel.close().sync();
            }
        } catch (Exception ignored) {
        }
        bossGroup.shutdownGracefully();
        workerGroup.shutdownGracefully();
    }

    // ============================================================
    // Netty ChannelHandler
    // ============================================================

    private static class ServerHandler extends SimpleChannelInboundHandler<FullHttpRequest> {

        private final Router router;

        ServerHandler(Router router) {
            this.router = router;
        }

        @Override
        protected void channelRead0(ChannelHandlerContext ctx, FullHttpRequest request) throws Exception {
            String path = sanitizePath(request.uri());
            String method = request.method().name();

            // 静态文件
            if (path.startsWith("/static/")) {
                serveStatic(ctx, path);
                return;
            }

            // 路由匹配
            Handler handler = router.match(method, path);
            if (handler != null) {
                handler.handle(ctx, request);
            } else {
                sendError(ctx, HttpResponseStatus.NOT_FOUND, "404 Not Found");
            }
        }

        @Override
        public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) {
            log.error("HTTP 处理异常", cause);
            ctx.close();
        }
    }

    // ============================================================
    // 静态文件
    // ============================================================

    private static void serveStatic(ChannelHandlerContext ctx, String path) {
        // 移除 /static/ 前缀
        String resourcePath = "static/" + path.substring(8);

        try (InputStream in = HttpServer.class.getClassLoader().getResourceAsStream(resourcePath)) {
            if (in == null) {
                sendError(ctx, HttpResponseStatus.NOT_FOUND, "404 Not Found");
                return;
            }

            byte[] data = in.readAllBytes();
            String contentType = getContentType(path);

            FullHttpResponse response = new DefaultFullHttpResponse(
                    HttpVersion.HTTP_1_1, HttpResponseStatus.OK,
                    Unpooled.wrappedBuffer(data));
            response.headers()
                    .set(HttpHeaderNames.CONTENT_TYPE, contentType)
                    .set(HttpHeaderNames.CONTENT_LENGTH, data.length)
                    .set(HttpHeaderNames.CACHE_CONTROL, "public, max-age=3600");
            ctx.writeAndFlush(response).addListener(ChannelFutureListener.CLOSE);
        } catch (Exception e) {
            sendError(ctx, HttpResponseStatus.INTERNAL_SERVER_ERROR, "读取静态文件失败");
        }
    }

    private static String getContentType(String path) {
        if (path.endsWith(".css")) return "text/css; charset=utf-8";
        if (path.endsWith(".js")) return "application/javascript; charset=utf-8";
        if (path.endsWith(".html")) return "text/html; charset=utf-8";
        if (path.endsWith(".png")) return "image/png";
        if (path.endsWith(".jpg") || path.endsWith(".jpeg")) return "image/jpeg";
        if (path.endsWith(".svg")) return "image/svg+xml";
        if (path.endsWith(".woff2")) return "font/woff2";
        if (path.endsWith(".woff")) return "font/woff";
        return "application/octet-stream";
    }

    // ============================================================
    // 工具方法
    // ============================================================

    public static void sendJson(ChannelHandlerContext ctx, int statusCode, String json) {
        FullHttpResponse response = new DefaultFullHttpResponse(
                HttpVersion.HTTP_1_1,
                HttpResponseStatus.valueOf(statusCode),
                Unpooled.copiedBuffer(json, CharsetUtil.UTF_8));
        response.headers()
                .set(HttpHeaderNames.CONTENT_TYPE, "application/json; charset=utf-8")
                .set(HttpHeaderNames.CONTENT_LENGTH, response.content().readableBytes());
        ctx.writeAndFlush(response).addListener(ChannelFutureListener.CLOSE);
    }

    public static void sendHtml(ChannelHandlerContext ctx, String html) {
        FullHttpResponse response = new DefaultFullHttpResponse(
                HttpVersion.HTTP_1_1, HttpResponseStatus.OK,
                Unpooled.copiedBuffer(html, CharsetUtil.UTF_8));
        response.headers()
                .set(HttpHeaderNames.CONTENT_TYPE, "text/html; charset=utf-8")
                .set(HttpHeaderNames.CONTENT_LENGTH, response.content().readableBytes());
        ctx.writeAndFlush(response).addListener(ChannelFutureListener.CLOSE);
    }

    public static void sendError(ChannelHandlerContext ctx, HttpResponseStatus status, String message) {
        String json = "{\"error\":\"" + escapeJson(message) + "\"}";
        sendJson(ctx, status.code(), json);
    }

    public static String sanitizePath(String uri) {
        int queryIdx = uri.indexOf('?');
        if (queryIdx >= 0) {
            return uri.substring(0, queryIdx);
        }
        return uri;
    }

    public static String getQueryParam(FullHttpRequest request, String key) {
        String uri = request.uri();
        int queryIdx = uri.indexOf('?');
        if (queryIdx < 0) return null;

        String query = uri.substring(queryIdx + 1);
        for (String part : query.split("&")) {
            String[] kv = part.split("=", 2);
            if (kv.length == 2 && kv[0].equals(key)) {
                return urlDecode(kv[1]);
            }
        }
        return null;
    }

    public static String getFormParam(FullHttpRequest request, String key) {
        String contentType = request.headers().get(HttpHeaderNames.CONTENT_TYPE);
        if (contentType == null || !contentType.contains("application/x-www-form-urlencoded")) {
            return null;
        }

        String body = request.content().toString(CharsetUtil.UTF_8);
        for (String part : body.split("&")) {
            String[] kv = part.split("=", 2);
            if (kv.length == 2 && kv[0].equals(key)) {
                return urlDecode(kv[1]);
            }
        }
        return null;
    }

    public static String getParam(FullHttpRequest request, String key) {
        String val = getFormParam(request, key);
        if (val != null) return val;
        return getQueryParam(request, key);
    }

    private static String escapeJson(String s) {
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

    private static String urlDecode(String s) {
        try {
            return java.net.URLDecoder.decode(s, CharsetUtil.UTF_8);
        } catch (Exception e) {
            return s;
        }
    }
}
