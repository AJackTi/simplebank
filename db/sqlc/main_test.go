package db

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

var testQueries *Queries
var testDB *sql.DB
var testDatabaseSkipReason string

func requireTestDatabase(t *testing.T) {
	t.Helper()

	if testDatabaseSkipReason != "" {
		t.Skip(testDatabaseSkipReason)
	}
}

func TestMain(m *testing.M) {
	dbDriver := os.Getenv("SIMPLEBANK_TEST_DB_DRIVER")
	if dbDriver == "" {
		dbDriver = "postgres"
	}
	dbSource := os.Getenv("SIMPLEBANK_TEST_DB_SOURCE")
	if dbSource == "" {
		dbSource = os.Getenv("DB_SOURCE")
	}
	if dbSource == "" {
		testDatabaseSkipReason = "database integration test skipped: set SIMPLEBANK_TEST_DB_SOURCE or DB_SOURCE to run it"
		os.Exit(m.Run())
	}

	var err error
	testDB, err = sql.Open(dbDriver, dbSource)
	if err != nil {
		log.Fatal("cannot connect to DB: ", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := testDB.PingContext(ctx); err != nil {
		log.Fatal("cannot ping DB: ", err)
	}

	testQueries = New(testDB)

	exitCode := m.Run()
	if err := testDB.Close(); err != nil {
		log.Printf("cannot close test DB: %v", err)
		if exitCode == 0 {
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}
