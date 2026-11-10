package pgx

import (
	"context"
	"crypto/tls"
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"

	"github.com/api-go/db"
	"github.com/api-go/safe"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
)

type SecurePgxConnector struct {
	conf *db.Config
	pwd  *safe.SafeBuf
	tls  *tls.Config
}

var databaseNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*$`)

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
	conn, err := tmpConnector.Connect(ctx)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "3D000" || !databaseNamePattern.MatchString(c.conf.DB) {
		return conn, err
	}

	maintenanceConfig := pgxConfig.Copy()
	maintenanceConfig.Database = "postgres"
	maintenanceConnector := stdlib.GetConnector(*maintenanceConfig)
	maintenanceConn, createErr := maintenanceConnector.Connect(ctx)
	if createErr != nil {
		return nil, err
	}
	execer, ok := maintenanceConn.(driver.ExecerContext)
	if !ok {
		_ = maintenanceConn.Close()
		return nil, err
	}
	_, createErr = execer.ExecContext(ctx, `CREATE DATABASE "`+c.conf.DB+`"`, nil)
	_ = maintenanceConn.Close()
	if createErr != nil {
		return nil, err
	}
	return tmpConnector.Connect(ctx)
}

func (c *SecurePgxConnector) Driver() driver.Driver {
	return stdlib.GetDefaultDriver()
}

func PgxConnector(conf *db.Config, pwd *safe.SafeBuf, tls *tls.Config) driver.Connector {
	return &SecurePgxConnector{conf: conf, pwd: pwd, tls: tls}
}

func init() {
	for _, typ := range []string{"pg", "pgsql", "postgres", "postgresql", "pgx"} {
		db.RegisterConnector(typ, PgxConnector)
	}
}
