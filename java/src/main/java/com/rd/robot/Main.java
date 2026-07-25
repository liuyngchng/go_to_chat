package com.gotochat;

import com.rd.robot.config.AppConfig;
import com.rd.robot.embedding.EmbeddingClient;
import com.rd.robot.kb.KnowledgeBaseManager;
import com.rd.robot.model.Config;
import com.rd.robot.server.HttpServer;
import com.rd.robot.server.Router;
import com.rd.robot.server.PageHandler;
import com.rd.robot.server.ChatHandler;
import com.rd.robot.server.VdbHandler;
import com.rd.robot.session.SessionManager;
import com.rd.robot.store.SQLiteStore;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;

public class Main {

    private static final Logger log = LoggerFactory.getLogger(Main.class);

    public static void main(String[] args) {
        // 1. 加载配置
        Config cfg = AppConfig.load("cfg.yml");
        log.info("启动对话机器人...");

        // 2. 设置日志级别
        System.setProperty("log.level", cfg.getServer().isDebug() ? "DEBUG" : "INFO");

        // 3. 检查 cfg.db 是否存在
        File dbFile = new File("cfg.db");
        if (!dbFile.exists()) {
            System.err.println("错误: cfg.db 不存在，请将 cfg.db.template 复制为 cfg.db 后重新启动");
            System.exit(1);
        }

        // 4. 初始化 SQLite 元数据存储
        SQLiteStore metaStore = new SQLiteStore("cfg.db");
        log.info("数据库初始化完成");

        // 5. 初始化会话管理器
        SessionManager sessionMgr = new SessionManager();

        // 6. 初始化 Embedding 客户端
        EmbeddingClient embClient = new EmbeddingClient(
                cfg.getApi().getEmbeddingApiUri(),
                cfg.getApi().getEmbeddingApiKey(),
                cfg.getApi().getEmbeddingModelName()
        );

        // 7. 初始化知识库管理器
        KnowledgeBaseManager kbManager = new KnowledgeBaseManager(cfg, metaStore, embClient);

        // 8. 启动后台文件处理 Worker
        kbManager.startFileWorker();

        // 9. 创建 Handler
        PageHandler pageHandler = new PageHandler(cfg);
        ChatHandler chatHandler = new ChatHandler(cfg, kbManager, sessionMgr, metaStore);
        VdbHandler vdbHandler = new VdbHandler(cfg, kbManager, metaStore);

        // 10. 创建路由表
        Router router = new Router();
        // 页面
        router.addRoute("GET", "/", pageHandler::index);
        router.addRoute("GET", "/vdb/idx", pageHandler::vdbIndex);
        // 聊天
        router.addRoute("POST", "/chat", chatHandler::chat);
        router.addRoute("POST", "/chat/clear", chatHandler::clear);
        // 知识库管理
        router.addRoute("POST", "/vdb/my/list", vdbHandler::myList);
        router.addRoute("POST", "/vdb/pub/list", vdbHandler::pubList);
        router.addRoute("POST", "/vdb/file/list", vdbHandler::fileList);
        router.addRoute("POST", "/vdb/set/default", vdbHandler::setDefault);
        router.addRoute("POST", "/vdb/create", vdbHandler::create);
        router.addRoute("POST", "/vdb/delete", vdbHandler::delete);
        router.addRoute("POST", "/vdb/upload", vdbHandler::upload);
        router.addRoute("POST", "/vdb/process/info", vdbHandler::processInfo);
        router.addRoute("POST", "/vdb/search", vdbHandler::search);
        router.addRoute("POST", "/vdb/file/delete", vdbHandler::fileDelete);

        // 11. 启动 Netty HTTP 服务器
        HttpServer server = new HttpServer(cfg.getServer().getPort(), router);
        server.start();

        // 12. 注册 JVM 关闭钩子
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            log.info("正在关闭服务...");
            kbManager.stopFileWorker();
            server.stop();
            metaStore.close();
            log.info("服务已关闭");
        }));

        log.info("服务启动 addr=:{}", cfg.getServer().getPort());
    }
}
