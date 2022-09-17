package logger

import (
	"os"
	logrus "github.com/sirupsen/logrus"
	m "streaming/pkg/models"
)

type Logger struct {
	*logrus.Logger
}

func New() *logrus.Logger {
	return logrus.New()
}

func NewLogger(debug bool, path string) *logrus.Logger {

	log := logrus.New()
	log.AddHook(&CliConn{})

	log.SetLevel(logrus.DebugLevel)
	if debug {
		log.SetFormatter(&logrus.TextFormatter{})
		log.Out = os.Stdout
		log.Debug("Дебаг")
	} else {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.Out = file
		} else {
			log.Fatalf("Ошибка при открытии файла для логов, файл=%s\n", path)
		}
		log.SetFormatter(&logrus.JSONFormatter{})
		log.Debug("Прод")
	}
	log.Info("_____NEW_START_____")
	return log
}

type CliConn struct {
	Ip string
	Peer m.Peer
}

func (h *CliConn) Levels() []logrus.Level {
    return []logrus.Level{logrus.InfoLevel}
}

func (h *CliConn) Fire(e *logrus.Entry) error {
    if _, ok := e.Data["clientInfo"]; !ok {
        e.Data["clientInfo"] = ""
    }
    return nil
}