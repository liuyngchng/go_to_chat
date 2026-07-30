package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class VectorConfig {
    @JsonProperty("backend")
    private String backend = "local";

    public String getBackend() { return backend; }
    public void setBackend(String backend) { this.backend = backend; }
}