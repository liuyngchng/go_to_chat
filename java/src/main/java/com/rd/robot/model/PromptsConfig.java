package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class PromptsConfig {
    @JsonProperty("chat_msg")
    private String chatMsg;

    public String getChatMsg() { return chatMsg; }
    public void setChatMsg(String chatMsg) { this.chatMsg = chatMsg; }
}
