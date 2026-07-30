package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class CreateUserRequest {
    @JsonProperty("user_name")
    private String userName;

    @JsonProperty("user_pwd")
    private String userPwd;

    private int role;
    private String note;

    public String getUserName() { return userName; }
    public void setUserName(String userName) { this.userName = userName; }
    public String getUserPwd() { return userPwd; }
    public void setUserPwd(String userPwd) { this.userPwd = userPwd; }
    public int getRole() { return role; }
    public void setRole(int role) { this.role = role; }
    public String getNote() { return note; }
    public void setNote(String note) { this.note = note; }
}
