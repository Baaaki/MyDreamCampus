package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

// Init initializes the global logger
func Init(env string) error {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
		// JSON encoding for Loki integration in production
		config.Encoding = "json"

		// Enable sampling to prevent log storms in production
		// Initial: Write first 3 identical logs
		// Thereafter: Write 1 out of every 100 identical logs after initial
		config.Sampling = &zap.SamplingConfig{
			Initial:    3,
			Thereafter: 100,
		}
	} else {
		config = zap.NewDevelopmentConfig()
		// Console encoding with colors for development
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
		// No sampling in development - see all logs
	}

	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1), // Skip wrapper functions to show actual caller
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	Log = logger
	return nil
}

// Convenience functions
func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	Log.Fatal(msg, fields...)
}

// Sync flushes any buffered log entries (should be called before app exit)
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// MustInit initializes logger or panics
func MustInit(env string) {
	if err := Init(env); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
}
