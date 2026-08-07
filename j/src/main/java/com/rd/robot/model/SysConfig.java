package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class SysConfig {
    private String name;

    @JsonProperty("auth")
    private boolean auth;

    @JsonProperty("api_auth")
    private boolean apiAuth = true;

    @JsonProperty("work_mode")
    private int workMode; // 0=KB, 1=CSM, 2=Dynamic

    @JsonProperty("default_workflow_id")
    private long defaultWorkflowId; // workflow ID for dynamic mode

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public boolean isAuth() { return auth; }
    public void setAuth(boolean auth) { this.auth = auth; }
    public boolean isApiAuth() { return apiAuth; }
    public void setApiAuth(boolean apiAuth) { this.apiAuth = apiAuth; }
    public int getWorkMode() { return workMode; }
    public void setWorkMode(int workMode) { this.workMode = workMode; }
    public long getDefaultWorkflowId() { return defaultWorkflowId; }
    public void setDefaultWorkflowId(long defaultWorkflowId) { this.defaultWorkflowId = defaultWorkflowId; }
}