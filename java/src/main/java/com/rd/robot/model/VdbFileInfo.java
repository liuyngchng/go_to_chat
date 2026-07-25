package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class VdbFileInfo {
    private long id;
    private String name;
    private String uid;
    @JsonProperty("vdb_id")
    private long vdbId;
    @JsonProperty("task_id")
    private String taskId;
    @JsonProperty("file_path")
    private String filePath;
    private double percent;
    @JsonProperty("process_info")
    private String processInfo;
    @JsonProperty("file_md5")
    private String fileMd5;
    @JsonProperty("create_time")
    private String createTime;

    public long getId() { return id; }
    public void setId(long id) { this.id = id; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getUid() { return uid; }
    public void setUid(String uid) { this.uid = uid; }
    public long getVdbId() { return vdbId; }
    public void setVdbId(long vdbId) { this.vdbId = vdbId; }
    public String getTaskId() { return taskId; }
    public void setTaskId(String taskId) { this.taskId = taskId; }
    public String getFilePath() { return filePath; }
    public void setFilePath(String filePath) { this.filePath = filePath; }
    public double getPercent() { return percent; }
    public void setPercent(double percent) { this.percent = percent; }
    public String getProcessInfo() { return processInfo; }
    public void setProcessInfo(String processInfo) { this.processInfo = processInfo; }
    public String getFileMd5() { return fileMd5; }
    public void setFileMd5(String fileMd5) { this.fileMd5 = fileMd5; }
    public String getCreateTime() { return createTime; }
    public void setCreateTime(String createTime) { this.createTime = createTime; }
}
