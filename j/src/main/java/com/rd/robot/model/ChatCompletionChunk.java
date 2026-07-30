package com.rd.robot.model;

import java.util.List;

public class ChatCompletionChunk {
    private List<Choice> choices;

    public List<Choice> getChoices() { return choices; }
    public void setChoices(List<Choice> choices) { this.choices = choices; }

    public static class Choice {
        private Delta delta;

        public Delta getDelta() { return delta; }
        public void setDelta(Delta delta) { this.delta = delta; }
    }

    public static class Delta {
        private String content;

        public String getContent() { return content; }
        public void setContent(String content) { this.content = content; }
    }
}
