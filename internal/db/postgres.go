package db

/*
import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func ConnectPostgres(ctx context.Context) (*pgx.Conn, error) {
	// Replace the connection string with your PostgreSQL connection string

	connString := os.Getenv("PG_URL")
	if connString == "" {
		connString = "postgres://postgres:mysecret@localhost:5432/insider_db"
	}

	// Create a new PostgreSQL connection pool
	config, err := pgx.ParseConfig(connString)
	if err != nil {
		panic(err)
	}
	config.RuntimeParams = map[string]string{"search_path": "public"} // Optional: set the search path to public

	pool, err := pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		panic(err)
	}
	//defer pool.Close(ctx)

	// Perform a simple query to test the connection
	var id int
	err = pool.QueryRow(context.Background(), "SELECT 1").Scan(&id)
	if err != nil {
		panic(err)
	}
	fmt.Println("Successfully connected to the PostgreSQL database.")
	return pool, nil
}
*/
