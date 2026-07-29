package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class FaqConfig {
    @JsonProperty("match_threshold")
    private double matchThreshold = 0.85;

    public double getMatchThreshold() { return matchThreshold; }
    public void setMatchThreshold(double matchThreshold) { this.matchThreshold = matchThreshold; }
}