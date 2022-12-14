package pzlog

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"strings"
	"time"
)

const (
	logTmFmt = "2006-01-02 15:04:05"
)

var (
	Logger *zap.Logger
	m      = map[string]interface{}{
		"debug":  zap.DebugLevel,
		"info":   zap.InfoLevel,
		"warn":   zap.WarnLevel,
		"error":  zap.ErrorLevel,
		"dpanic": zap.DPanicLevel,
		"panic":  zap.PanicLevel,
		"fatal":  zap.FatalLevel,
	}
)

type PzlogConfig struct {
	lumberjack.Logger

	TimeFormat string `json:"timeformat" yaml:"timeformat"`

	LogLevel string `json:"loglevel" yaml:"loglevel"`

	PrintConsole bool `json:"printconsole" yaml:"printconsole"`

	// 日志格式，json或者console
	Encoder string `json:"encoder" yaml:"encoder"`
}

func NewDefaultConfig() *PzlogConfig {
	return &PzlogConfig{
		Logger: lumberjack.Logger{
			Filename:   "./logs/pzlog.log",
			MaxSize:    100,
			MaxBackups: 10,
			MaxAge:     30,
		},
		TimeFormat:   "",
		LogLevel:     "",
		PrintConsole: false,
		Encoder:      "",
	}
}

func setDefaultValue(config *PzlogConfig) {

	if config.Filename == "" {
		config.Filename = "./logs/pzlog.log"
	}
	if config.TimeFormat == "" {
		config.TimeFormat = logTmFmt
	}
	if config.MaxSize < 0 {
		config.MaxSize = 100
	}
	if config.MaxBackups < 0 {
		config.MaxBackups = 10
	}
	if config.MaxAge < 0 {
		config.MaxAge = 30
	}
	if config.Encoder == "" {
		config.Encoder = "json"
	}
	_, ok := m[strings.ToLower(config.LogLevel)]
	if config.LogLevel == "" || !ok {
		config.LogLevel = "info"
	}

}

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()
		cost := time.Since(start)
		zap.L().Info(path,
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			zap.Duration("cost", cost),
		)
	}
}

func GetLogger(config *PzlogConfig) *zap.Logger {
	if config == nil {
		config = NewDefaultConfig()
	}
	setDefaultValue(config)
	Encoder := getEncoder(config.Encoder)
	WriteSyncer := getWriteSyncer(config)
	LevelEnabler := getLevelEnabler(config)
	//ConsoleEncoder := getConsoleEncoder(config.Encoder)
	var newCore zapcore.Core
	if config.PrintConsole {
		newCore = zapcore.NewTee(
			zapcore.NewCore(Encoder, WriteSyncer, LevelEnabler),             // 写入文件
			zapcore.NewCore(Encoder, zapcore.Lock(os.Stdout), LevelEnabler), // 写入控制台
		)
	} else {
		newCore = zapcore.NewCore(Encoder, WriteSyncer, LevelEnabler)
	}
	return zap.New(newCore, zap.AddCaller())
}

// GetEncoder 自定义的Encoder
func getEncoder(types string) zapcore.Encoder {
	if types == "console" {
		return zapcore.NewConsoleEncoder(
			zapcore.EncoderConfig{
				TimeKey:        "ts",
				LevelKey:       "level",
				NameKey:        "logger",
				CallerKey:      "caller_line",
				FunctionKey:    zapcore.OmitKey,
				MessageKey:     "msg",
				StacktraceKey:  "stacktrace",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    cEncodeLevel,
				EncodeTime:     cEncodeTime,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   cEncodeCaller,
			})
	} else {
		return zapcore.NewJSONEncoder(
			zapcore.EncoderConfig{
				TimeKey:        "ts",
				LevelKey:       "level",
				NameKey:        "logger",
				CallerKey:      "caller_line",
				FunctionKey:    zapcore.OmitKey,
				MessageKey:     "msg",
				StacktraceKey:  "stacktrace",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    cEncodeLevel,
				EncodeTime:     cEncodeTime,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   cEncodeCaller,
			})
	}

}

// getConsoleEncoder 输出日志到控制台
func getConsoleEncoder(types string) zapcore.Encoder {
	if types == "console" {
		return zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	}
	return zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
}

// getWriteSyncer 自定义的WriteSyncer
func getWriteSyncer(config *PzlogConfig) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
	}
	return zapcore.AddSync(lumberJackLogger)
}

// GetLevelEnabler 自定义的LevelEnabler
func getLevelEnabler(config *PzlogConfig) zapcore.Level {
	level := strings.ToLower(config.LogLevel)
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "dpanic":
		return zap.DPanicLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}

// cEncodeLevel 自定义日志级别显示
func cEncodeLevel(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(level.CapitalString())
}

// cEncodeTime 自定义时间格式显示
func cEncodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(logTmFmt))
}

// cEncodeCaller 自定义行号显示
func cEncodeCaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(caller.TrimmedPath())
}
