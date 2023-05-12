package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.SugaredLogger

func Init(config *Config) {
	config = withDefaultConf(config)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	level, err := zapcore.ParseLevel(config.StdoutLevel)
	if err != nil {
		level = zapcore.InfoLevel
	}

	cores := make([]zapcore.Core, 1)
	cores[0] = zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), level)

	if config.Filename != "" {
		level, err = zapcore.ParseLevel(config.Level)
		if err != nil {
			level = zapcore.ErrorLevel
		}

		priority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
			return lev >= level
		})
		infoFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   config.Filename,
			MaxSize:    config.MaxSize,
			MaxAge:     config.MaxAge,
			MaxBackups: config.MaxBackups,
			Compress:   config.Compress,
		})

		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		core := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), infoFileWriteSyncer, priority)
		cores = append(cores, core)
	}

	logger = zap.New(zapcore.NewTee(cores...)).Sugar()
}

func Debug(args ...interface{}) {
	logger.Debug(args...)
}

func Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	logger.Warnf(template, args...)
}

func Error(args ...interface{}) {
	logger.Error(args...)
}

func Errorf(template string, args ...interface{}) {
	logger.Errorf(template, args...)
}

func Panicf(template string, args ...interface{}) {
	logger.Panicf(template, args...)
}

func Sync() {
	logger.Sync()
}

func Shutdown() {
	Info("log shutdown...")
	Sync()
}
