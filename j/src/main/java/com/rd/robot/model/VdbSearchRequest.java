package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class VdbSearchRequest {
    @JsonProperty("vdb_id")
    private Long vdbId;

    @JsonProperty("vdb_ids")
    private List<Long> vdbIds;

    private String query;

    public Long getVdbId() { return vdbId; }
    public void setVdbId(Long vdbId) { this.vdbId = vdbId; }
    public List<Long> getVdbIds() { return vdbIds; }
    public void setVdbIds(List<Long> vdbIds) { this.vdbIds = vdbIds; }
    public String getQuery() { return query; }
    public void setQuery(String query) { this.query = query; }
}
