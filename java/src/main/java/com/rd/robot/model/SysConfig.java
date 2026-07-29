package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class SysConfig {
    private String name;

    @JsonProperty("auth")
    private boolean auth;

    @JsonProperty("api_auth")
    private boolean apiAuth = true;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public boolean isAuth() { return auth; }
    public void setAuth(boolean auth) { this.auth = auth; }
    public boolean isApiAuth() { return apiAuth; }
    public void setApiAuth(boolean apiAuth) { this.apiAuth = apiAuth; }
}