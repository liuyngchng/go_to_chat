package com.rd.robot.model;

public class Config {
    private ServerConfig server;
    private SysConfig sys;
    private APIConfig api;
    private MilvusConfig milvus;
    private PromptsConfig prompts;

    public ServerConfig getServer() { return server; }
    public void setServer(ServerConfig server) { this.server = server; }
    public SysConfig getSys() { return sys; }
    public void setSys(SysConfig sys) { this.sys = sys; }
    public APIConfig getApi() { return api; }
    public void setApi(APIConfig api) { this.api = api; }
    public MilvusConfig getMilvus() { return milvus; }
    public void setMilvus(MilvusConfig milvus) { this.milvus = milvus; }
    public PromptsConfig getPrompts() { return prompts; }
    public void setPrompts(PromptsConfig prompts) { this.prompts = prompts; }
}
