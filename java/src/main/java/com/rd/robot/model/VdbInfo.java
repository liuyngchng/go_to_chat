package com.rd.robot.model;

public class VdbInfo {
    private long id;
    private String name;
    private String uid;
    private boolean isPublic;
    private boolean isDefault;
    private String createTime;

    public long getId() { return id; }
    public void setId(long id) { this.id = id; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getUid() { return uid; }
    public void setUid(String uid) { this.uid = uid; }
    public boolean isPublic() { return isPublic; }
    public void setPublic(boolean isPublic) { this.isPublic = isPublic; }
    public boolean isDefault() { return isDefault; }
    public void setDefault(boolean isDefault) { this.isDefault = isDefault; }
    public String getCreateTime() { return createTime; }
    public void setCreateTime(String createTime) { this.createTime = createTime; }
}
