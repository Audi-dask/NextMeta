package configs

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

/*
Config 对应 config.yaml 的根配置结构。
服务启动时会通过 LoadConfig 读取该结构，并把 server、database、jwt
三类配置分发给 HTTP 服务、数据库连接和 JWT 模块使用。
*/
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
}

/*
JWTConfig 对应 config.yaml 里的 jwt 配置段。
Expires 和 RefreshExpires 保留原始字符串配置，ExpiresDuration 和
RefreshExpiresDuration 是解析后的运行时值，不参与 YAML 反序列化。
*/
type JWTConfig struct {
	SecretKey      string `yaml:"secret_key"`
	Expires        string `yaml:"expires"`
	RefreshExpires string `yaml:"refresh_expires"`

	ExpiresDuration        time.Duration `yaml:"-"`
	RefreshExpiresDuration time.Duration `yaml:"-"`
}

/*
ServerConfig 对应 config.yaml 里的 server 配置段。
当前只保存 HTTP 服务监听端口，cmd/server/main.go 会使用该端口启动 Gin 服务。
*/
type ServerConfig struct {
	Port string `yaml:"port"`
}

/*
DatabaseConfig 对应 config.yaml 里的 database 配置段。
后端主服务和维护工具都会基于这些字段拼接 MySQL DSN，用于连接业务数据库。
*/
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

/*
GetDSN 将数据库配置拼接成 GORM MySQL Driver 可识别的 DSN。
DSN 中固定启用 utf8mb4、时间字段解析和本地时区，保证业务库读写行为一致。
*/
func (c *DatabaseConfig) GetDSN() string {
	// DSN format: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.DBName)
}

/*
parseDuration 解析配置里的时间长度字符串。
Go 原生 time.ParseDuration 不支持 d 作为天单位，这里额外支持 7d 这类配置，
其余格式仍交给 time.ParseDuration 处理，例如 1h、30m、10s。
*/
func parseDuration(value string) (time.Duration, error) {
	if strings.HasSuffix(value, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(value, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(value)
}

/*
ApplyDefaults 为 JWT 配置补齐默认有效期，并转换成运行时使用的 time.Duration。
当配置文件未显式设置 access token 或 refresh token 有效期时，分别默认使用
1h 和 7d；解析失败会返回错误，由 LoadConfig 在启动阶段中止服务。
*/
func (c *JWTConfig) ApplyDefaults() error {
	if strings.TrimSpace(c.SecretKey) == "" {
		return fmt.Errorf("jwt.secret_key is required")
	}
	if c.Expires == "" {
		c.Expires = "1h"
	}
	if c.RefreshExpires == "" {
		c.RefreshExpires = "7d"
	}

	expires, err := parseDuration(c.Expires)
	if err != nil {
		return fmt.Errorf("invalid jwt.expires: %w", err)
	}
	refreshExpires, err := parseDuration(c.RefreshExpires)
	if err != nil {
		return fmt.Errorf("invalid jwt.refresh_expires: %w", err)
	}

	c.ExpiresDuration = expires
	c.RefreshExpiresDuration = refreshExpires
	return nil
}

/*
LoadConfig 读取并解析当前工作目录下的 config.yaml。
配置文件缺失、YAML 解析失败或 JWT 时间配置非法都会直接终止进程，
避免服务在缺少关键配置的情况下继续启动。
*/
func LoadConfig() *Config {
	// Try to load from config.yaml
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Printf("警告：无法读取connect.yaml, 使用默认值失败: %v", err)
		// For safety during migration, we could return defaults, but the goal is to move to YAML.
		// Let's fail if config is missing to ensure we are using the file.
		log.Fatalf("读取时出错 config.yaml: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("解析错误 config.yaml: %v", err)
	}
	if err := config.JWT.ApplyDefaults(); err != nil {
		log.Fatalf("解析错误 JWT config: %v", err)
	}
	return &config
}
