package configs

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

/*
Config 对应 config.yaml 的根配置结构。
*/
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type JWTConfig struct {
	Secret  string `yaml:"secret"`
	Expires int    `yaml:"expires"` // 单位：分钟
	Refresh int    `yaml:"refresh"` // 单位：分钟
}

/*
LoadConfig 读取并解析当前工作目录下的 config.yaml。
配置文件缺失或 YAML 解析失败时返回错误，由启动入口统一记录日志并决定是否退出。
*/
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("read config.yaml: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config.yaml: %w", err)
	}

	return &config, nil
}
