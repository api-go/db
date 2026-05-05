package db

import (
	"apigo.cc/go/cast"
	"apigo.cc/go/log"
)

type DBLog struct {
	log.BaseLog
	DbType     string
	Dsn        string
	Query      string
	QueryArgs  string
	UsedTime   float32
	Error      string
	CallStacks []string
}

func (dl *dbLogger) LogDB(query string, args []any, usedTime float32, err error, extra ...any) {
	LogDB(dl.logger, dl.config, query, args, usedTime, err, extra...)
}

func LogDB(logger *log.Logger, conf *Config, query string, args []any, usedTime float32, err error, extra ...any) {
	if logger == nil {
		return
	}

	logType := log.LogTypeDb
	level := log.INFO
	var e string
	if err != nil {
		logType = log.LogTypeDbError
		level = log.ERROR
		e = err.Error()
	}

	if logger.CheckLevel(level) {
		entry := log.GetEntry[DBLog]()
		// 仅关注业务字段，LogType 手动赋值，基础字段由 logger.Log 自动填充
		entry.LogType = logType
		entry.DbType = conf.Type
		entry.Dsn = conf.Dsn()
		entry.Query = query
		entry.QueryArgs = cast.To[string](args)
		entry.UsedTime = usedTime
		if e != "" {
			entry.Error = e
			entry.CallStacks = logger.GetCallStacks()
		}
		if len(extra) > 0 {
			cast.FillMap(&entry.Extra, extra)
		}
		logger.Log(entry)
	}
}
