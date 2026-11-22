package main

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConnectDBRequiresConfiguration(t *testing.T) {
	for _, key := range []string{"ORDER_DB_HOST", "ORDER_DB_PORT", "ORDER_DB_NAME", "ORDER_DB_USER", "ORDER_DB_PASSWORD_FILE"} {
		t.Setenv(key, "")
	}
	for _, value := range []string{"", " \t\n"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("ORDER_DATABASE_URL", value)
			db, err := connectDB(context.Background())
			if db != nil {
				db.Close()
				t.Fatal("expected no database connection without configuration")
			}
			if err == nil || !strings.Contains(err.Error(), "ORDER_DATABASE_URL is required") {
				t.Fatalf("expected missing configuration error, got %v", err)
			}
		})
	}
}

func TestOrderDatabaseDSN(t *testing.T) {
	t.Setenv("ORDER_DATABASE_URL", "")
	t.Setenv("ORDER_DB_HOST", "order-db")
	t.Setenv("ORDER_DB_PORT", "5432")
	t.Setenv("ORDER_DB_NAME", "order_db")
	t.Setenv("ORDER_DB_USER", "order_user")
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("p@ss:/?#% word\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ORDER_DB_PASSWORD_FILE", path)
	dsn, err := orderDatabaseDSN()
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal("invalid DSN")
	}
	password, ok := u.User.Password()
	if !ok || password != "p@ss:/?#% word" || u.Host != "order-db:5432" || u.Path != "/order_db" || u.User.Username() != "order_user" {
		t.Fatal("incorrect connection parameters")
	}
	t.Setenv("ORDER_DATABASE_URL", " postgres://local/db ")
	t.Setenv("ORDER_DB_PASSWORD_FILE", "missing")
	if got, err := orderDatabaseDSN(); err != nil || got != "postgres://local/db" {
		t.Fatal("URL should take precedence")
	}
	t.Setenv("ORDER_DATABASE_URL", "")
	if _, err := orderDatabaseDSN(); err == nil || strings.Contains(err.Error(), "missing") {
		t.Fatal("expected safe password file error")
	}
}

func TestPaymentTLSServerName(t *testing.T) {
	t.Setenv("PAYMENT_TLS_SERVER_NAME", "")
	if got := paymentTLSServerName(); got != "localhost" {
		t.Fatalf("default server name = %q", got)
	}
	t.Setenv("PAYMENT_TLS_SERVER_NAME", "payment")
	if got := paymentTLSServerName(); got != "payment" {
		t.Fatalf("configured server name = %q", got)
	}
}
