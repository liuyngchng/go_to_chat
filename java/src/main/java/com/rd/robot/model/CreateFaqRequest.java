package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class CreateFaqRequest {
    private List<String> questions;
    private String answer;

    public List<String> getQuestions() { return questions; }
    public void setQuestions(List<String> questions) { this.questions = questions; }
    public String getAnswer() { return answer; }
    public void setAnswer(String answer) { this.answer = answer; }
}