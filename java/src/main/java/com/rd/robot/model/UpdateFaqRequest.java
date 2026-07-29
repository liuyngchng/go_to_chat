package com.rd.robot.model;

import java.util.List;

public class UpdateFaqRequest {
    private List<String> questions;
    private String answer;

    public List<String> getQuestions() { return questions; }
    public void setQuestions(List<String> questions) { this.questions = questions; }
    public String getAnswer() { return answer; }
    public void setAnswer(String answer) { this.answer = answer; }
}