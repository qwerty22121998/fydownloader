package log

import (
	"go.uber.org/zap"
)

func init() {
	conf := zap.NewDevelopmentConfig()
	//conf.OutputPaths = []string{
	//	"fydownloader.log",
	//}
	logger, _ := conf.Build()
	zap.ReplaceGlobals(logger)
}

func S() *zap.SugaredLogger {
	return zap.S()
}
