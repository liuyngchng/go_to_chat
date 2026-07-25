package com.rd.robot.server;

import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;

@FunctionalInterface
public interface Handler {
    void handle(ChannelHandlerContext ctx, FullHttpRequest request) throws Exception;
}
