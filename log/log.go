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

func initLogger(logPath string, level string)(err error) {
	var cfg zap.Config
	//var err error
	var logt *zap.Logger

	cfg.Level = level
	cfg.Encoding = "json"
	cfg.OutputPaths = logPath
	cfg.ErrorOutputPaths = logPath
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
		return nil
	} else {
		Log.Info(fmt.Sprintf("[rename logstatsh ok]the file %s\n",oldfile))
	}
	logB := Log
	err = initLogger(logfile, level)
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
		var time3 int64 = int64((60-mnow) * time.Minute)
		time.Sleep(time.Duration(time3))
		tm := time.Unix(now, 0)
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


