package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ResetPwdRequest {
    @JsonProperty("user_pwd")
    private String userPwd;

    public String getUserPwd() { return userPwd; }
    public void setUserPwd(String userPwd) { this.userPwd = userPwd; }
}
