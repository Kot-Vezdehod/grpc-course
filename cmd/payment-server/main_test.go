package main

import (
	"context"
	"testing"
)

func TestConnectDBRequiresConfiguration(t *testing.T) {
	for _, value := range []string{"", " \t\n"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("PAYMENT_DATABASE_URL", value)
			db, err := connectDB(context.Background())
			if db != nil {
				db.Close()
				t.Fatal("expected no database connection without configuration")
			}
			if err == nil || err.Error() != "PAYMENT_DATABASE_URL is required" {
				t.Fatalf("expected missing configuration error, got %v", err)
			}
		})
	}
}
