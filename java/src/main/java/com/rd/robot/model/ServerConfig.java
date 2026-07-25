package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ServerConfig {
    private int port;
    @JsonProperty("debug")
    private boolean debug;

    public int getPort() { return port; }
    public void setPort(int port) { this.port = port; }
    public boolean isDebug() { return debug; }
    public void setDebug(boolean debug) { this.debug = debug; }
}
