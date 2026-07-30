package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class VdbSearchRequest {
    @JsonProperty("vdb_id")
    private long vdbId;

    private String query;

    public long getVdbId() { return vdbId; }
    public void setVdbId(long vdbId) { this.vdbId = vdbId; }
    public String getQuery() { return query; }
    public void setQuery(String query) { this.query = query; }
}
