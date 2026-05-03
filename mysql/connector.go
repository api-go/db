package mysql

import (
	"context"
	"crypto/tls"
	"database/sql/driver"

	"github.com/go-sql-driver/mysql"
	"apigo.cc/go/db"
	"apigo.cc/go/safe"
)

type SecureMysqlConnector struct {
	conf *db.Config
	pwd  *safe.SafeBuf
	tls  *tls.Config
}

func (c *SecureMysqlConnector) Connect(ctx context.Context) (driver.Conn, error) {
	cfg, err := mysql.ParseDSN(c.conf.Args)
	if err != nil {
		cfg = mysql.NewConfig()
	}
	cfg.User = c.conf.User
	cfg.Net = "tcp"
	cfg.Addr = c.conf.Host
	cfg.DBName = c.conf.DB
	cfg.TLS = c.tls
	pwdBuf := c.pwd.Open()
	defer pwdBuf.Close()
	cfg.Passwd = pwdBuf.String()

	tmpConnector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, err
	}
	return tmpConnector.Connect(ctx)
}

func (c *SecureMysqlConnector) Driver() driver.Driver {
	return &mysql.MySQLDriver{}
}

func init() {
	db.RegisterConnector("mysql", func(conf *db.Config, pwd *safe.SafeBuf, tls *tls.Config) driver.Connector {
		return &SecureMysqlConnector{conf: conf, pwd: pwd, tls: tls}
	})
}
