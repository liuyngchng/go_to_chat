package com.rd.robot.web.router;

import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;

/**
 * Route handler functional interface.
 * Same signature as the old Handler.java.
 */
@FunctionalInterface
public interface RouteHandler {
    void handle(ChannelHandlerContext ctx, FullHttpRequest request) throws Exception;
}