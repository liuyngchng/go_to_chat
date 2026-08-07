package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class Config {
    @JsonProperty("server")
    private ServerConfig server;

    @JsonProperty("sys")
    private SysConfig sys;

    @JsonProperty("api")
    private APIConfig api;

    @JsonProperty("store")
    private StoreConfig store;

    @JsonProperty("vector")
    private VectorConfig vector;

    @JsonProperty("milvus")
    private MilvusConfig milvus;

    @JsonProperty("qdrant")
    private QdrantConfig qdrant;

    @JsonProperty("mysql")
    private MySQLConfig mysql;

    @JsonProperty("kb")
    private KBConfig kb;

    @JsonProperty("llm")
    private LLMParams llm;

    @JsonProperty("redis")
    private RedisConfig redis;

    @JsonProperty("oss")
    private OSSConfig oss;

    @JsonProperty("faq")
    private FaqConfig faq;

    @JsonProperty("prompts")
    private PromptsConfig prompts;

    public ServerConfig getServer() { return server; }
    public void setServer(ServerConfig server) { this.server = server; }
    public SysConfig getSys() { return sys; }
    public void setSys(SysConfig sys) { this.sys = sys; }
    public APIConfig getApi() { return api; }
    public void setApi(APIConfig api) { this.api = api; }
    public StoreConfig getStore() { return store; }
    public void setStore(StoreConfig store) { this.store = store; }
    public VectorConfig getVector() { return vector; }
    public void setVector(VectorConfig vector) { this.vector = vector; }
    public MilvusConfig getMilvus() { return milvus; }
    public void setMilvus(MilvusConfig milvus) { this.milvus = milvus; }
    public QdrantConfig getQdrant() { return qdrant; }
    public void setQdrant(QdrantConfig qdrant) { this.qdrant = qdrant; }
    public MySQLConfig getMysql() { return mysql; }
    public void setMysql(MySQLConfig mysql) { this.mysql = mysql; }
    public RedisConfig getRedis() { return redis; }
    public void setRedis(RedisConfig redis) { this.redis = redis; }
    public OSSConfig getOss() { return oss; }
    public void setOss(OSSConfig oss) { this.oss = oss; }
    public KBConfig getKb() { return kb; }
    public void setKb(KBConfig kb) { this.kb = kb; }
    public LLMParams getLlm() { return llm; }
    public void setLlm(LLMParams llm) { this.llm = llm; }
    public FaqConfig getFaq() { return faq; }
    public void setFaq(FaqConfig faq) { this.faq = faq; }
    public PromptsConfig getPrompts() { return prompts; }
    public void setPrompts(PromptsConfig prompts) { this.prompts = prompts; }
}