package com.rd.robot.web.router;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.regex.Pattern;

/**
 * Router with path parameter support.
 * Supports patterns like /api/vdb/:id/files, /api/faq/:id, etc.
 */
public class Router {

    // Static routes: method:path -> handler
    private final Map<String, RouteHandler> staticRoutes = new ConcurrentHashMap<>();

    // Parametric routes: method -> list of (pattern, paramNames, handler)
    private final Map<String, List<ParamRoute>> paramRoutes = new ConcurrentHashMap<>();

    /**
     * Add a route. Supports :param placeholders in the path.
     */
    public void addRoute(String method, String path, RouteHandler handler) {
        if (path.contains(":")) {
            // Parametric route
            String[] segments = path.split("/");
            List<String> paramNames = new ArrayList<>();
            StringBuilder regexBuilder = new StringBuilder();
            for (String seg : segments) {
                if (seg.isEmpty()) continue;
                if (seg.startsWith(":")) {
                    paramNames.add(seg.substring(1));
                    regexBuilder.append("/([^/]+)");
                } else {
                    regexBuilder.append("/").append(Pattern.quote(seg));
                }
            }

            ParamRoute pr = new ParamRoute(Pattern.compile(regexBuilder.toString()), paramNames, handler);
            paramRoutes.computeIfAbsent(method.toUpperCase(), k -> new ArrayList<>()).add(pr);
        } else {
            // Static route
            staticRoutes.put(method.toUpperCase() + ":" + path, handler);
        }
    }

    /**
     * Match a route and return the handler.
     * Sets path parameters as request attributes via the context.
     */
    public RouteMatch match(String method, String path) {
        String key = method.toUpperCase() + ":" + path;

        // Try static first
        RouteHandler handler = staticRoutes.get(key);
        if (handler != null) {
            return new RouteMatch(handler, Map.of());
        }

        // Try parametric
        List<ParamRoute> paramList = paramRoutes.get(method.toUpperCase());
        if (paramList != null) {
            for (ParamRoute pr : paramList) {
                java.util.regex.Matcher m = pr.pattern.matcher(path);
                if (m.matches()) {
                    Map<String, String> params = new HashMap<>();
                    for (int i = 0; i < pr.paramNames.size(); i++) {
                        params.put(pr.paramNames.get(i), m.group(i + 1));
                    }
                    return new RouteMatch(pr.handler, params);
                }
            }
        }

        return null;
    }

    /**
     * Route match result with extracted path parameters.
     */
    public static class RouteMatch {
        private final RouteHandler handler;
        private final Map<String, String> pathParams;

        RouteMatch(RouteHandler handler, Map<String, String> pathParams) {
            this.handler = handler;
            this.pathParams = pathParams;
        }

        public RouteHandler getHandler() { return handler; }
        public Map<String, String> getPathParams() { return pathParams; }
    }

    private static class ParamRoute {
        final Pattern pattern;
        final List<String> paramNames;
        final RouteHandler handler;

        ParamRoute(Pattern pattern, List<String> paramNames, RouteHandler handler) {
            this.pattern = pattern;
            this.paramNames = paramNames;
            this.handler = handler;
        }
    }
}