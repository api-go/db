package db

import "testing"

func TestNetworkDatabaseDefaults(t *testing.T) {
	tests := []struct {
		setting string
		typ     string
		host    string
		dbName  string
		user    string
	}{
		{"mysql", "mysql", "127.0.0.1:3306", "apigo", "root"},
		{"mysql://", "mysql", "127.0.0.1:3306", "apigo", "root"},
		{"mysql://dbhost/example", "mysql", "dbhost:3306", "example", "root"},
		{"postgres", "postgres", "127.0.0.1:5432", "postgres", ""},
		{"postgresql://dbhost/example", "postgresql", "dbhost:5432", "example", ""},
	}
	for _, tt := range tests {
		t.Run(tt.setting, func(t *testing.T) {
			conf := new(Config)
			conf.ConfigureBy(tt.setting)
			if conf.Type != tt.typ || conf.Host != tt.host || conf.DB != tt.dbName {
				t.Fatalf("ConfigureBy(%q) = type %q, host %q, db %q", tt.setting, conf.Type, conf.Host, conf.DB)
			}
			if tt.user != "" && conf.User != tt.user {
				t.Fatalf("ConfigureBy(%q) user = %q, want %q", tt.setting, conf.User, tt.user)
			}
			if tt.setting == "mysql" {
				if localUser, ok := conf.LocalAuthFallback(); !ok || localUser == "" {
					t.Fatal("mysql shorthand should permit local socket authentication fallback")
				}
			}
			if tt.setting == "mysql://dbhost/example" {
				if _, ok := conf.LocalAuthFallback(); ok {
					t.Fatal("explicit MySQL host must not permit authentication fallback")
				}
			}
			if isPostgresType(conf.Type) && conf.User == "" {
				t.Fatalf("ConfigureBy(%q) should default to the current OS user", tt.setting)
			}
		})
	}
}
