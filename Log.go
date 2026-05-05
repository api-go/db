package db

import (
	"apigo.cc/go/cast"
	"apigo.cc/go/log"
)

type DBLog struct {
	DbType    string
	Dsn       string
	Query     string
	QueryArgs string
	UsedTime  float32
}

type DBInfoLog struct {
	log.InfoLog
	DBLog
}

type DBErrorLog struct {
	log.ErrorLog
	DBLog
}

func (dl *dbLogger) LogDB(query string, args []any, usedTime float32, err error, extra ...any) {
	LogDB(dl.logger, dl.config, query, args, usedTime, err, extra...)
}

func LogDB(logger *log.Logger, conf *Config, query string, args []any, usedTime float32, err error, extra ...any) {
	if logger == nil {
		return
	}

	if err != nil {
		if logger.CheckLevel(log.ERROR) {
			entry := log.GetEntry[DBErrorLog]()
			entry.LogType = log.LogTypeDbError
			entry.Error = err.Error()
			entry.DBLog = DBLog{
				DbType:    conf.Type,
				Dsn:       conf.Dsn(),
				Query:     query,
				QueryArgs: cast.To[string](args),
				UsedTime:  usedTime,
			}
			if len(extra) > 0 {
				cast.FillMap(&entry.Extra, extra)
			}
			logger.Log(entry)
		}
	} else {
		if logger.CheckLevel(log.INFO) {
			entry := log.GetEntry[DBInfoLog]()
			entry.LogType = log.LogTypeDb
			entry.DBLog = DBLog{
				DbType:    conf.Type,
				Dsn:       conf.Dsn(),
				Query:     query,
				QueryArgs: cast.To[string](args),
				UsedTime:  usedTime,
			}
			if len(extra) > 0 {
				cast.FillMap(&entry.Extra, extra)
			}
			logger.Log(entry)
		}
	}
}
