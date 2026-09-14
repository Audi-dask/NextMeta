package logger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const slowSQLThreshold = 200 * time.Millisecond

// Log 是项目全局 Zap logger。
var Log *zap.Logger

// eventEncoder 将 Zap message 收敛为结构化 event 字段。
type eventEncoder struct {
	zapcore.Encoder
}

func (e eventEncoder) Clone() zapcore.Encoder {
	return eventEncoder{Encoder: e.Encoder.Clone()}
}

func (e eventEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	if entry.Message != "" {
		fields = append([]zapcore.Field{zap.String("event", entry.Message)}, fields...)
	}
	entry.Message = ""
	return e.Encoder.EncodeEntry(entry, fields)
}

// mysqlLogger 将 MySQL driver 的内部错误接入项目日志出口。
type mysqlLogger struct{}

func (mysqlLogger) Print(values ...any) {
	Log.Error("mysql_driver_error", zap.String("error", fmt.Sprint(values...)))
}

// InitLogger 初始化全局 JSON 日志器，并接管 MySQL driver 日志。
func InitLogger() {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.LevelKey = "level"
	encoderConfig.CallerKey = "caller"
	encoderConfig.MessageKey = ""
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	encoderConfig.EncodeDuration = zapcore.MillisDurationEncoder

	encoder := eventEncoder{Encoder: zapcore.NewJSONEncoder(encoderConfig)}
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
	Log = zap.New(core, zap.AddCaller())

	if err := mysqlDriver.SetLogger(mysqlLogger{}); err != nil {
		Log.Error("mysql_logger_setup_failed", zap.Error(err))
	}
}

type gormZapLogger struct {
	level gormlogger.LogLevel
}

func (l *gormZapLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	clone := *l
	clone.level = level
	return &clone
}

func (l *gormZapLogger) ParamsFilter(_ context.Context, sql string, _ ...interface{}) (string, []interface{}) {
	return sql, nil
}

func (l *gormZapLogger) Info(_ context.Context, message string, values ...interface{}) {
	if l.level >= gormlogger.Info {
		Log.Info("gorm_info", zap.String("detail", fmt.Sprintf(message, values...)))
	}
}

func (l *gormZapLogger) Warn(_ context.Context, message string, values ...interface{}) {
	if l.level >= gormlogger.Warn {
		Log.Warn("gorm_warning", zap.String("detail", fmt.Sprintf(message, values...)))
	}
}

func (l *gormZapLogger) Error(_ context.Context, message string, values ...interface{}) {
	if l.level >= gormlogger.Error {
		Log.Error("gorm_error", zap.String("detail", fmt.Sprintf(message, values...)))
	}
}

func (l *gormZapLogger) Trace(_ context.Context, begin time.Time, query func() (string, int64), err error) {
	if l.level == gormlogger.Silent || errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}

	latency := time.Since(begin)
	if err != nil && l.level >= gormlogger.Error {
		sql, rows := query()
		Log.Error("gorm_query_failed",
			zap.Error(err),
			zap.String("sql", sql),
			zap.Int64("rows_affected", rows),
			zap.Float64("latency_ms", float64(latency.Microseconds())/1000),
		)
		return
	}

	if latency >= slowSQLThreshold && l.level >= gormlogger.Warn {
		sql, rows := query()
		Log.Warn("gorm_slow_query",
			zap.String("sql", sql),
			zap.Int64("rows_affected", rows),
			zap.Float64("latency_ms", float64(latency.Microseconds())/1000),
		)
	}
}

// GormConfig 返回接入项目 Zap 日志的统一 GORM 配置。
func GormConfig() *gorm.Config {
	return &gorm.Config{
		Logger: &gormZapLogger{level: gormlogger.Error},
	}
}
