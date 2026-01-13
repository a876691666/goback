package logger

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/goback/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	once      sync.Once
	defLogger *Logger
)

// Logger 日志管理器
type Logger struct {
	*zap.Logger
	sugar *zap.SugaredLogger
}

// Init 初始化日志
func Init(cfg *config.LogConfig) error {
	var err error
	once.Do(func() {
		defLogger, err = newLogger(cfg)
	})
	return err
}

// newLogger 创建日志实例
func newLogger(cfg *config.LogConfig) (*Logger, error) {
	// 解析日志级别
	level := parseLevel(cfg.Level)

	// 创建编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建编码器
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 创建输出
	var cores []zapcore.Core

	switch cfg.Output {
	case "console":
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level))
	case "file":
		cores = append(cores, zapcore.NewCore(encoder, getFileWriter(cfg), level))
	case "both":
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level))
		cores = append(cores, zapcore.NewCore(encoder, getFileWriter(cfg), level))
	default:
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level))
	}

	// 创建logger
	core := zapcore.NewTee(cores...)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))

	return &Logger{
		Logger: zapLogger,
		sugar:  zapLogger.Sugar(),
	}, nil
}

// parseLevel 解析日志级别
func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// getFileWriter 获取文件写入器
func getFileWriter(cfg *config.LogConfig) zapcore.WriteSyncer {
	// 确保日志目录存在
	dir := filepath.Dir(cfg.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}

	// 使用lumberjack进行日志轮转
	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	})
}

// Get 获取日志实例
func Get() *Logger {
	if defLogger == nil {
		// 默认初始化
		defLogger, _ = newLogger(&config.LogConfig{
			Level:  "debug",
			Format: "console",
			Output: "console",
		})
	}
	return defLogger
}

// Sugar 获取SugaredLogger
func Sugar() *zap.SugaredLogger {
	return Get().sugar
}

// Sync 同步日志
func Sync() error {
	if defLogger != nil {
		return defLogger.Logger.Sync()
	}
	return nil
}

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	Get().Logger.Debug(msg, fields...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	Get().Logger.Info(msg, fields...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	Get().Logger.Warn(msg, fields...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	Get().Logger.Error(msg, fields...)
}

// Fatal 致命错误日志
func Fatal(msg string, fields ...zap.Field) {
	Get().Logger.Fatal(msg, fields...)
}

// Debugf 格式化调试日志
func Debugf(template string, args ...interface{}) {
	Sugar().Debugf(template, args...)
}

// Infof 格式化信息日志
func Infof(template string, args ...interface{}) {
	Sugar().Infof(template, args...)
}

// Warnf 格式化警告日志
func Warnf(template string, args ...interface{}) {
	Sugar().Warnf(template, args...)
}

// Errorf 格式化错误日志
func Errorf(template string, args ...interface{}) {
	Sugar().Errorf(template, args...)
}

// Fatalf 格式化致命错误日志
func Fatalf(template string, args ...interface{}) {
	Sugar().Fatalf(template, args...)
}

// WithFields 添加字段
func WithFields(fields ...zap.Field) *zap.Logger {
	return Get().Logger.With(fields...)
}

// WithContext 添加上下文字段
func WithContext(ctx map[string]interface{}) *zap.Logger {
	fields := make([]zap.Field, 0, len(ctx))
	for k, v := range ctx {
		fields = append(fields, zap.Any(k, v))
	}
	return Get().Logger.With(fields...)
}

// Field 创建字段
func Field(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

// String 创建字符串字段
func String(key string, value string) zap.Field {
	return zap.String(key, value)
}

// Int 创建整数字段
func Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

// Int64 创建64位整数字段
func Int64(key string, value int64) zap.Field {
	return zap.Int64(key, value)
}

// Err 创建错误字段
func Err(err error) zap.Field {
	return zap.Error(err)
}

// Duration 创建时长字段
func Duration(key string, value int64) zap.Field {
	return zap.Int64(key, value)
}
