package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.SugaredLogger

func Init(conf *Config) {
	conf = withDefaultConf(conf)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	level, err := zapcore.ParseLevel(conf.StdoutLevel)
	if err != nil {
		level = zapcore.InfoLevel
	}

	cores := make([]zapcore.Core, 1)
	cores[0] = zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), level)

	if conf.Filename != "" {
		level, err = zapcore.ParseLevel(conf.Level)
		if err != nil {
			level = zapcore.ErrorLevel
		}

		priority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
			return lev >= level
		})
		infoFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   conf.Filename,
			MaxSize:    conf.MaxSize,
			MaxAge:     conf.MaxAge,
			MaxBackups: conf.MaxBackups,
			Compress:   conf.Compress,
		})

		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		core := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), infoFileWriteSyncer, priority)
		cores = append(cores, core)
	}

	logger = zap.New(zapcore.NewTee(cores...)).Sugar()
}

func Panicf(template string, args ...interface{}) {
	logger.Panicf(template, args...)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	logger.Debugw(msg, keysAndValues...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	logger.Infow(msg, keysAndValues...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	logger.Warnw(msg, keysAndValues...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	logger.Errorw(msg, keysAndValues...)
}

func Panicw(msg string, keysAndValues ...interface{}) {
	logger.Panicw(msg, keysAndValues...)
}

func Sync() {
	logger.Sync()
}

func Shutdown() {
	Infow("log shutdown...")
	Sync()
}
