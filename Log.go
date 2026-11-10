package db

import (
	"github.com/api-go/cast"
	"github.com/api-go/log"
)

type DBLog struct {
	DbType    string  `log:"pos:7,color:blue"`
	Dsn       string  `log:"pos:8,color:gray,withoutkey:true"`
	Query     string  `log:"pos:9,color:cyan"`
	QueryArgs string  `log:"pos:10,color:gray"`
	UsedTime  float32 `log:"pos:11,format:%.2fms"`
}

func (l *DBLog) Reset() {
	l.DbType = ""
	l.Dsn = ""
	l.Query = ""
	l.QueryArgs = ""
	l.UsedTime = 0
}

type DBInfoLog struct {
	log.InfoLog
	DBLog
}

func (l *DBInfoLog) Reset() {
	l.InfoLog.Reset()
	l.DBLog.Reset()
}

type DBErrorLog struct {
	log.ErrorLog
	DBLog
}

func (l *DBErrorLog) Reset() {
	l.ErrorLog.Reset()
	l.DBLog.Reset()
}

func init() {
	log.RegisterType(log.LogTypeDb, DBInfoLog{})
	log.RegisterType(log.LogTypeDbError, DBErrorLog{})
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
