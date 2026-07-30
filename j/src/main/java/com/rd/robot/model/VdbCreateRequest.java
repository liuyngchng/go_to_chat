package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class VdbCreateRequest {
    private String name;

    @JsonProperty("is_public")
    private boolean isPublic;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public boolean isPublic() { return isPublic; }
    public void setPublic(boolean isPublic) { this.isPublic = isPublic; }
}
