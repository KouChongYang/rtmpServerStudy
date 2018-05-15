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

func initLogger(logPath string, level string) (Log *zap.Logger){

	var cfg zap.Config
	var err error

	cfg.Level = level
	cfg.Encoding = "json"
	cfg.OutputPaths = logPath
	cfg.ErrorOutputPaths = logPath
	cfg.EncoderConfig = zap.NewProductionEncoderConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	Log, err = cfg.Build()
	if err != nil {
		log.Fatal("init logger error: ", err)
	}
	return Log
}

func ReloadLoger(oldfile ,logfile ,level string) (Log *zap.Logger) {

	err := os.Rename(logfile, oldfile)
	if err != nil {
		return nil
	} else {
		Log.Info(fmt.Sprintf("[rename logstatsh ok]the file %s\n",))
	}

	return initLogger(logfile, level)
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
		ReloadLoger
	}
}

func main() {
	Log:=initLogger("out.log", "INFO", false)

	s := []string{
		"hello info",
		"hello error",
		"hello debug",
		"hello fatal",
	}

	Log.Info("info:", zap.String("s", s[0]))
	Log.Error("info:", zap.String("s", s[1]))
	Log.Debug("info:", zap.String("s", s[2]))
	Log.Fatal("info:", zap.String("s", s[3]))
}




