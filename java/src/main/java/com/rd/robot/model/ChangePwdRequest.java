package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ChangePwdRequest {
    @JsonProperty("old_pwd")
    private String oldPwd;

    @JsonProperty("new_pwd")
    private String newPwd;

    public String getOldPwd() { return oldPwd; }
    public void setOldPwd(String oldPwd) { this.oldPwd = oldPwd; }
    public String getNewPwd() { return newPwd; }
    public void setNewPwd(String newPwd) { this.newPwd = newPwd; }
}
