package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.LocalDateTime;

public class ApiCallLog {
    private long id;

    @JsonProperty("user_name")
    private String userName;

    @JsonProperty("api_path")
    private String apiPath;

    private String method;

    @JsonProperty("request_body")
    private String requestBody;

    @JsonProperty("response_body")
    private String responseBody;

    @JsonProperty("status_code")
    private int statusCode;

    @JsonProperty("error_msg")
    private String errorMsg;

    @JsonProperty("create_time")
    private LocalDateTime createTime;

    public long getId() { return id; }
    public void setId(long id) { this.id = id; }
    public String getUserName() { return userName; }
    public void setUserName(String userName) { this.userName = userName; }
    public String getApiPath() { return apiPath; }
    public void setApiPath(String apiPath) { this.apiPath = apiPath; }
    public String getMethod() { return method; }
    public void setMethod(String method) { this.method = method; }
    public String getRequestBody() { return requestBody; }
    public void setRequestBody(String requestBody) { this.requestBody = requestBody; }
    public String getResponseBody() { return responseBody; }
    public void setResponseBody(String responseBody) { this.responseBody = responseBody; }
    public int getStatusCode() { return statusCode; }
    public void setStatusCode(int statusCode) { this.statusCode = statusCode; }
    public String getErrorMsg() { return errorMsg; }
    public void setErrorMsg(String errorMsg) { this.errorMsg = errorMsg; }
    public LocalDateTime getCreateTime() { return createTime; }
    public void setCreateTime(LocalDateTime createTime) { this.createTime = createTime; }
}