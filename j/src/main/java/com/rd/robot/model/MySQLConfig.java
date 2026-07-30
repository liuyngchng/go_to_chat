package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class MySQLConfig {
    @JsonProperty("dsn")
    private String dsn;

    public String getDsn() { return dsn; }
    public void setDsn(String dsn) { this.dsn = dsn; }
}