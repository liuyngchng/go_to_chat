package config

import (
	"fmt"
	"os"

	"go_to_chat/internal/model"

	"gopkg.in/yaml.v3"
)

// Load 从文件加载配置
func Load(path string) (*model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}

	var cfg model.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 19007
	}

	return &cfg, nil
}
