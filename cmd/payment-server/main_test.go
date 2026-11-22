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
	for _, key := range []string{"PAYMENT_DB_HOST", "PAYMENT_DB_PORT", "PAYMENT_DB_NAME", "PAYMENT_DB_USER", "PAYMENT_DB_PASSWORD_FILE"} {
		t.Setenv(key, "")
	}
	for _, value := range []string{"", " \t\n"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("PAYMENT_DATABASE_URL", value)
			db, err := connectDB(context.Background())
			if db != nil {
				db.Close()
				t.Fatal("expected no database connection without configuration")
			}
			if err == nil || !strings.Contains(err.Error(), "PAYMENT_DATABASE_URL is required") {
				t.Fatalf("expected missing configuration error, got %v", err)
			}
		})
	}
}

func TestPaymentDatabaseDSN(t *testing.T) {
	t.Setenv("PAYMENT_DATABASE_URL", "")
	t.Setenv("PAYMENT_DB_HOST", "payment-db")
	t.Setenv("PAYMENT_DB_PORT", "5432")
	t.Setenv("PAYMENT_DB_NAME", "payment_db")
	t.Setenv("PAYMENT_DB_USER", "payment_user")
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("p@ss:/?#% word\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAYMENT_DB_PASSWORD_FILE", path)
	dsn, err := paymentDatabaseDSN()
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal("invalid DSN")
	}
	password, ok := u.User.Password()
	if !ok || password != "p@ss:/?#% word" || u.Host != "payment-db:5432" || u.Path != "/payment_db" || u.User.Username() != "payment_user" {
		t.Fatal("incorrect connection parameters")
	}
	t.Setenv("PAYMENT_DATABASE_URL", " postgres://local/db ")
	t.Setenv("PAYMENT_DB_PASSWORD_FILE", "missing")
	if got, err := paymentDatabaseDSN(); err != nil || got != "postgres://local/db" {
		t.Fatal("URL should take precedence")
	}
	t.Setenv("PAYMENT_DATABASE_URL", "")
	if _, err := paymentDatabaseDSN(); err == nil || strings.Contains(err.Error(), "missing") {
		t.Fatal("expected safe password file error")
	}
}

func TestConnectDBDoesNotExposeDSN(t *testing.T) {
	t.Setenv("PAYMENT_DATABASE_URL", "postgres://user:secret@%invalid/db")
	_, err := connectDB(context.Background())
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatal("expected a sanitized connection error")
	}
}
