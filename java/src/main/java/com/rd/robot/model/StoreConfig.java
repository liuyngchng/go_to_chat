package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class StoreConfig {
    @JsonProperty("backend")
    private String backend = "sqlite";

    public String getBackend() { return backend; }
    public void setBackend(String backend) { this.backend = backend; }
}