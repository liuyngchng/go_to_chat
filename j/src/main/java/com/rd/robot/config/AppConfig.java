package com.rd.robot.config;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import com.rd.robot.model.Config;

import java.io.File;

public class AppConfig {

    public static Config load(String path) {
        try {
            ObjectMapper mapper = new ObjectMapper(new YAMLFactory());
            Config cfg = mapper.readValue(new File(path), Config.class);

            if (cfg.getServer().getMode() == null || cfg.getServer().getMode().isEmpty()) {
                cfg.getServer().setMode("singleton");
            }
            if (cfg.getServer().getRole() == null || cfg.getServer().getRole().isEmpty()) {
                cfg.getServer().setRole("all");
            }
            if (cfg.getServer().getPort() == 0) {
                cfg.getServer().setPort(19007);
            }

            return cfg;
        } catch (Exception e) {
            throw new RuntimeException("加载配置文件失败: " + path, e);
        }
    }
}
