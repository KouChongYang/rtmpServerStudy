package log

import (
	"log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
	"fmt"
	"os"
)

var Log *zap.Logger

func InitLogger(logPath string, level string)(err error) {
	var cfg zap.Config
	//var err error
	var logt *zap.Logger

	//zap.DebugLevel
	switch level {
	case "debug", "DEBUG":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info", "INFO", "": // make the zero value useful
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn", "WARN":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error", "ERROR":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "dpanic", "DPANIC":
		cfg.Level = zap.NewAtomicLevelAt(zap.DPanicLevel)
	case "panic", "PANIC":
		cfg.Level = zap.NewAtomicLevelAt(zap.PanicLevel)
	case "fatal", "FATAL":
		cfg.Level = zap.NewAtomicLevelAt(zap.FatalLevel)
	default:
		return fmt.Errorf("bad level")
	}

	cfg.Encoding = "json"
	cfg.OutputPaths = []string{logPath}
	cfg.ErrorOutputPaths = []string{logPath}
	cfg.EncoderConfig = zap.NewProductionEncoderConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logB := Log
	logt, err = cfg.Build()
	if err != nil {
		log.Fatal("init logger error: ", err)
		Log = logB
	}else{
		Log = logt
	}

	return
}

func ReloadLoger(oldfile ,logfile ,level string)  {

	err := os.Rename(logfile, oldfile)
	if err != nil {
		return
	} else {
		Log.Info(fmt.Sprintf("[rename logstatsh ok]the file %s\n",oldfile))
	}
	logB := Log
	err = InitLogger(logfile, level)
	if err != nil {
		Log.Info(fmt.Sprintf("[rename logstatsh ok]the file %s\n",logfile))
	}else{
		Log = logB
	}

	return
}

func logCutByHour(LogPath,level string) {

	for {
		now := time.Now()
		mnow := now.Minute()
		//var time3 int64 = int64((60-mnow) * time.Minute)
		time.Sleep(time.Duration(60-mnow) * time.Minute)
		tm := time.Unix(now.Unix(),0)
		tim1 := tm.Format("2006-01-02-15")
		path_name := fmt.Sprintf("%s%s", LogPath, tim1)
		ReloadLoger(path_name,LogPath,level)
	}
}

func Info(msg string, fields ...zap.Field){
	Log.Info(msg,fields...)
}

func Debug(msg string, fields ...zap.Field){
	Log.Debug(msg,fields...)
}

func Error(msg string, fields ...zap.Field){
	Log.Error(msg,fields...)
}


