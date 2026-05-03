package pgx

import (
	"context"
	"crypto/tls"
	"database/sql/driver"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"apigo.cc/go/db"
	"apigo.cc/go/safe"
)

type SecurePgxConnector struct {
	conf *db.Config
	pwd  *safe.SafeBuf
	tls  *tls.Config
}

func (c *SecurePgxConnector) Connect(ctx context.Context) (driver.Conn, error) {
	dsn := fmt.Sprintf("postgres://%s@%s/%s?%s", c.conf.User, c.conf.Host, c.conf.DB, c.conf.Args)
	pgxConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	if c.tls != nil {
		pgxConfig.TLSConfig = c.tls
	}
	pwdBuf := c.pwd.Open()
	defer pwdBuf.Close()
	pgxConfig.Password = pwdBuf.String()
	tmpConnector := stdlib.GetConnector(*pgxConfig)
	return tmpConnector.Connect(ctx)
}

func (c *SecurePgxConnector) Driver() driver.Driver {
	return stdlib.GetDefaultDriver()
}

func PgxConnector(conf *db.Config, pwd *safe.SafeBuf, tls *tls.Config) driver.Connector {
	return &SecurePgxConnector{conf: conf, pwd: pwd, tls: tls}
}

func init() {
	db.RegisterConnector("postgres", PgxConnector)
	db.RegisterConnector("pgx", PgxConnector)
}
