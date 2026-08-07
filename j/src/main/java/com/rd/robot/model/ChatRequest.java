package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ChatRequest {
    @JsonProperty("msg")
    private String msg;

    @JsonProperty("uid")
    private String uid;

    @JsonProperty("app_source")
    private String appSource;

    public String getMsg() { return msg; }
    public void setMsg(String msg) { this.msg = msg; }
    public String getUid() { return uid; }
    public void setUid(String uid) { this.uid = uid; }
    public String getAppSource() { return appSource; }
    public void setAppSource(String appSource) { this.appSource = appSource; }
}