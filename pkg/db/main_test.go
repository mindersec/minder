package db

import (
	"log"
	"os"
	"testing"

	"database/sql"

	_ "github.com/lib/pq"
)

var testQueries *Queries

func TestMain(m *testing.M) {
	connStr := "user=postgres dbname=postgres password=postgres host=localhost sslmode=disable"
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("cannot connect to db test instance:", err)
	}

	testQueries = New(conn)

	// Run tests
	os.Exit(m.Run())
}
