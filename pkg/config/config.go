package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var (
	once   sync.Once
	config *Config
)

// Config 全局配置结构
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Etcd     EtcdConfig     `mapstructure:"etcd"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Casbin   CasbinConfig   `mapstructure:"casbin"`
	Log      LogConfig      `mapstructure:"log"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`
	Version string `mapstructure:"version"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	HTTP HTTPConfig `mapstructure:"http"`
	GRPC GRPCConfig `mapstructure:"grpc"`
}

// HTTPConfig HTTP服务配置
type HTTPConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"readTimeout"`
	WriteTimeout int    `mapstructure:"writeTimeout"`
}

// GRPCConfig GRPC服务配置
type GRPCConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver       string `mapstructure:"driver"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Database     string `mapstructure:"database"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"maxIdleConns"`
	MaxOpenConns int    `mapstructure:"maxOpenConns"`
	LogLevel     string `mapstructure:"logLevel"`
}

// DSN 生成数据库连接字符串
func (c *DatabaseConfig) DSN() string {
	switch c.Driver {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
			c.Username, c.Password, c.Host, c.Port, c.Database, c.Charset)
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			c.Host, c.Port, c.Username, c.Password, c.Database)
	case "sqlite":
		// SQLite 直接使用文件路径作为 DSN
		// 如果 Database 为空或为 ":memory:"，则使用内存数据库
		if c.Database == "" {
			return ":memory:"
		}
		return c.Database
	default:
		return ""
	}
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"poolSize"`
	Mode     string `mapstructure:"mode"` // "standalone" 外部 Redis, "memory" 内存模式
}

// Addr 获取Redis地址
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// EtcdConfig Etcd配置
type EtcdConfig struct {
	Endpoints   []string `mapstructure:"endpoints"`
	DialTimeout int      `mapstructure:"dialTimeout"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Issuer string `mapstructure:"issuer"`
	Expire int64  `mapstructure:"expire"`
}

// CasbinConfig Casbin配置
type CasbinConfig struct {
	ModelPath string `mapstructure:"modelPath"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"maxSize"`
	MaxBackups int    `mapstructure:"maxBackups"`
	MaxAge     int    `mapstructure:"maxAge"`
	Compress   bool   `mapstructure:"compress"`
}

// Init 初始化配置
func Init(configPath string) error {
	var err error
	once.Do(func() {
		config = &Config{}
		err = loadConfig(configPath)
	})
	return err
}

// loadConfig 加载配置文件
func loadConfig(configPath string) error {
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath("../configs")
		v.AddConfigPath("../../configs")
	}

	// 读取环境变量
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 加载环境特定配置
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = v.GetString("app.env")
	}

	if env != "" && env != "default" {
		v.SetConfigName(fmt.Sprintf("config.%s", env))
		if err := v.MergeInConfig(); err != nil {
			// 环境配置文件不存在不报错
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return fmt.Errorf("failed to merge env config: %w", err)
			}
		}
	}

	// 解析配置到结构体
	if err := v.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 处理环境变量占位符
	resolveEnvVars(config)

	return nil
}

// resolveEnvVars 解析环境变量占位符
func resolveEnvVars(cfg *Config) {
	cfg.Database.Host = resolveEnvVar(cfg.Database.Host)
	cfg.Database.Username = resolveEnvVar(cfg.Database.Username)
	cfg.Database.Password = resolveEnvVar(cfg.Database.Password)
	cfg.Database.Database = resolveEnvVar(cfg.Database.Database)
	cfg.Redis.Host = resolveEnvVar(cfg.Redis.Host)
	cfg.Redis.Password = resolveEnvVar(cfg.Redis.Password)
	cfg.JWT.Secret = resolveEnvVar(cfg.JWT.Secret)
}

// resolveEnvVar 解析单个环境变量
func resolveEnvVar(value string) string {
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		envKey := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
		if envValue := os.Getenv(envKey); envValue != "" {
			return envValue
		}
	}
	return value
}

// Get 获取配置实例
func Get() *Config {
	if config == nil {
		panic("config not initialized, call Init first")
	}
	return config
}

// GetApp 获取应用配置
func GetApp() *AppConfig {
	return &Get().App
}

// GetServer 获取服务器配置
func GetServer() *ServerConfig {
	return &Get().Server
}

// GetDatabase 获取数据库配置
func GetDatabase() *DatabaseConfig {
	return &Get().Database
}

// GetRedis 获取Redis配置
func GetRedis() *RedisConfig {
	return &Get().Redis
}

// GetJWT 获取JWT配置
func GetJWT() *JWTConfig {
	return &Get().JWT
}

// GetLog 获取日志配置
func GetLog() *LogConfig {
	return &Get().Log
}

// IsDev 是否为开发环境
func IsDev() bool {
	return Get().App.Env == "dev" || Get().App.Env == "development"
}

// IsProd 是否为生产环境
func IsProd() bool {
	return Get().App.Env == "prod" || Get().App.Env == "production"
}
