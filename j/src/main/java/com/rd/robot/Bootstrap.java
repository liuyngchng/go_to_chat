package com.rd.robot;

import com.rd.robot.client.ClientFactory;
import com.rd.robot.config.AppConfig;
import com.rd.robot.config.RuntimeConfig;
import com.rd.robot.knowledge.KnowledgeBaseManager;
import com.rd.robot.model.Config;
import com.rd.robot.repository.MetaStore;
import com.rd.robot.repository.MysqlMetaStore;
import com.rd.robot.repository.SqliteMetaStore;
import com.rd.robot.session.SessionManager;
import com.rd.robot.web.controller.*;
import com.rd.robot.web.router.Router;
import com.rd.robot.web.server.HttpServer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;

/**
 * Application entry point.
 */
public class Bootstrap {

    private static final Logger log = LoggerFactory.getLogger(Bootstrap.class);

    public static void main(String[] args) {
        // 1. Load config
        Config cfg = AppConfig.load("cfg.yml");
        log.info("启动对话机器人...");

        // 2. Set log level
        System.setProperty("log.level", cfg.getServer().isDebug() ? "DEBUG" : "INFO");

        // 3. Initialize metadata store
        MetaStore metaStore = createMetaStore(cfg);
        log.info("数据库初始化完成");

        // 4. Load runtime config from DB
        RuntimeConfig.load(metaStore, cfg);
        log.info("运行时配置加载完成");

        // 5. Initialize session manager (TODO: 后续迁移至 Redis 替代 SQLite 持久化)
        SessionManager sessionMgr = new SessionManager(metaStore);

        // 6. Initialize client factory (lazy, reads config from DB)
        ClientFactory clientFactory = new ClientFactory(metaStore);

        // 7. Initialize knowledge base manager
        KnowledgeBaseManager kbManager = new KnowledgeBaseManager(cfg, metaStore, clientFactory);

        // 8. Start background file processing worker
        kbManager.startFileWorker();

        // 9. Create controllers
        PageController pageController = new PageController(cfg);
        AuthController authController = new AuthController(metaStore);
        FaqController faqController = new FaqController(metaStore, clientFactory);
        ChatController chatController = new ChatController(cfg, kbManager, sessionMgr, metaStore, clientFactory, faqController);
        VdbController vdbController = new VdbController(cfg, kbManager, metaStore, chatController.getCsmEngine());
        ConfigController configController = new ConfigController(cfg, metaStore, clientFactory);
        UserController userController = new UserController(metaStore);
        AgentController agentController = new AgentController(metaStore);
        WorkflowController workflowController = new WorkflowController(metaStore);

        // 10. Create router and register routes
        Router router = new Router();

        // -- Health --
        router.addRoute("GET", "/health", (ctx, req) -> {
            HttpServer.sendJson(ctx, 200, "{\"status\":\"ok\"}");
        });

        // -- Pages --
        router.addRoute("GET", "/login", pageController::loginPage);
        router.addRoute("GET", "/", pageController::index);
        router.addRoute("GET", "/vdb/idx", pageController::vdbIndex);
        router.addRoute("GET", "/admin/config", pageController::configIndex);
        router.addRoute("GET", "/admin/vdb/bind", pageController::vdbBindIndex);
        router.addRoute("GET", "/user/api", pageController::userApiIndex);

        // -- Auth API (JSON) --
        router.addRoute("POST", "/api/login", authController::login);
        router.addRoute("POST", "/api/logout", authController::logout);
        router.addRoute("GET", "/api/me", authController::me);
        router.addRoute("GET", "/api/agents", authController::getOnlineAgents);

        // -- Chat API --
        router.addRoute("POST", "/api/chat", chatController::chat);
        router.addRoute("POST", "/api/chat/sync", chatController::chatSync);
        router.addRoute("GET", "/api/chat/history", chatController::history);
        router.addRoute("POST", "/api/chat/clear", chatController::clear);

        // -- Config API --
        router.addRoute("GET", "/api/config", configController::getConfig);
        router.addRoute("PUT", "/api/config", configController::updateConfig);
        router.addRoute("POST", "/api/config/test-models", configController::testModels);
        router.addRoute("GET", "/api/info", configController::info);

        // -- VDB API --
        router.addRoute("GET", "/api/vdb", vdbController::myList);
        router.addRoute("GET", "/api/vdb/pub", vdbController::pubList);
        router.addRoute("POST", "/api/vdb", vdbController::create);
        router.addRoute("DELETE", "/api/vdb/:id", vdbController::delete);
        router.addRoute("PUT", "/api/vdb/:id/default", vdbController::setDefault);
        router.addRoute("GET", "/api/vdb/:id/files", vdbController::fileList);
        router.addRoute("POST", "/api/vdb/:id/upload", vdbController::upload);
        router.addRoute("POST", "/api/vdb/search", vdbController::search);
        router.addRoute("GET", "/api/vdb/file/:id/progress", vdbController::processInfo);
        router.addRoute("GET", "/api/vdb/file/:id/chunks", vdbController::chunks);
        router.addRoute("GET", "/api/vdb/file/:id/download", vdbController::download);
        router.addRoute("DELETE", "/api/vdb/file/:id", vdbController::fileDelete);
        router.addRoute("GET", "/api/vdb/bindings", vdbController::bindingGet);
        router.addRoute("PUT", "/api/vdb/bindings", vdbController::bindingPut);

        // -- FAQ API --
        router.addRoute("GET", "/api/faq", faqController::list);
        router.addRoute("GET", "/api/faq/template", faqController::template);
        router.addRoute("POST", "/api/faq/match", faqController::match);
        router.addRoute("POST", "/api/faq", faqController::create);
        router.addRoute("POST", "/api/faq/upload", faqController::upload);
        router.addRoute("PUT", "/api/faq/:id", faqController::update);
        router.addRoute("DELETE", "/api/faq/:id", faqController::delete);
        router.addRoute("DELETE", "/api/faq", faqController::clearAll);

        // -- User API (admin) --
        router.addRoute("GET", "/api/users", userController::listUsers);
        router.addRoute("POST", "/api/users", userController::createUser);
        router.addRoute("DELETE", "/api/users/:name", userController::deleteUser);
        router.addRoute("PUT", "/api/users/:name/reset-pwd", userController::resetUserPwd);

        // -- User API (self-service) --
        router.addRoute("PUT", "/api/user/password", userController::changePassword);
        router.addRoute("GET", "/api/user/tokens", userController::listMyTokens);
        router.addRoute("POST", "/api/user/token", userController::generateToken);
        router.addRoute("GET", "/api/user/call-logs", userController::myCallLogs);

        // -- AI Agent API --
        router.addRoute("GET", "/api/system-vars", agentController::listSystemVars);
        router.addRoute("GET", "/api/ai-agents/public", agentController::listPublic);
        router.addRoute("GET", "/api/ai-agents", agentController::list);
        router.addRoute("POST", "/api/ai-agents", agentController::create);
        router.addRoute("GET", "/api/ai-agents/:id", agentController::get);
        router.addRoute("PUT", "/api/ai-agents/:id", agentController::update);
        router.addRoute("DELETE", "/api/ai-agents/:id", agentController::delete);

        // -- Workflow API --
        router.addRoute("GET", "/api/workflows", workflowController::listPublic);
        router.addRoute("POST", "/api/workflows", workflowController::create);
        router.addRoute("GET", "/api/workflows/:id", workflowController::get);
        router.addRoute("PUT", "/api/workflows/:id", workflowController::update);
        router.addRoute("DELETE", "/api/workflows/:id", workflowController::delete);

        // -- Classifier test --
        router.addRoute("POST", "/api/classifier/test", chatController::testClassifier);

        // 11. Start HTTP server
        HttpServer server = new HttpServer(cfg.getServer().getPort(), router, cfg);
        server.start();

        // 12. Register shutdown hook
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            log.info("正在关闭服务...");
            kbManager.stopFileWorker();
            server.stop();
            metaStore.close();
            log.info("服务已关闭");
        }));

        log.info("服务启动: http://localhost:{}", cfg.getServer().getPort());
    }

    private static MetaStore createMetaStore(Config cfg) {
        String backend = cfg.getStore() != null ? cfg.getStore().getBackend() : "sqlite";

        if ("mysql".equals(backend)) {
            if (cfg.getMysql() == null || cfg.getMysql().getDsn() == null || cfg.getMysql().getDsn().isEmpty()) {
                System.err.println("错误: store.backend=mysql 但 mysql.dsn 为空");
                System.exit(1);
            }
            log.info("使用 MySQL 存储");
            return new MysqlMetaStore(cfg.getMysql().getDsn());
        }

        // SQLite default
        if (!new File("cfg.db").exists()) {
            System.err.println("错误: cfg.db 不存在，请将 cfg.db.template 复制为 cfg.db 后重新启动");
            System.exit(1);
        }
        log.info("使用 SQLite 存储");
        return new SqliteMetaStore("cfg.db");
    }
}