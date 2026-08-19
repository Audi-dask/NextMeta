package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

/*
Log 是项目全局 zap logger。
业务代码需要记录日志时优先使用该对象，而不是直接 fmt.Println。
*/
var Log *zap.Logger

/*
InitLogger 初始化全局日志器。
当前日志输出到标准输出，使用 ISO8601 时间格式、大写日志级别，并记录调用方位置。
*/
func InitLogger() {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
	)

	Log = zap.New(core, zap.AddCaller())
}

/*
GormConfig 返回项目统一的 GORM 配置。
仅输出 Error 级别日志，避免记录慢 SQL 和源码路径。
*/
func GormConfig() *gorm.Config {
	return &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Error),
	}
}
