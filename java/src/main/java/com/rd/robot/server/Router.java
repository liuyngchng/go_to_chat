package com.rd.robot.server;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

public class Router {

    private final Map<String, Handler> routes = new ConcurrentHashMap<>();

    public void addRoute(String method, String path, Handler handler) {
        routes.put(method.toUpperCase() + ":" + path, handler);
    }

    public Handler match(String method, String path) {
        return routes.get(method.toUpperCase() + ":" + path);
    }
}
