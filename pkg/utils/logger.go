package utils

import (
	"os"

	zaplogfmt "github.com/sykesm/zap-logfmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger() *zap.SugaredLogger {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = ""

	logger := zap.New(zapcore.NewCore(
		zaplogfmt.NewEncoder(config),
		os.Stdout,
		zapcore.InfoLevel,
	))

	return logger.Sugar()
}
