package repository

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"goacs/lib"
	"time"
)

var connection *sqlx.DB

func InitConnection() *sqlx.DB {
	var env *lib.Env = new(lib.Env)

	fmt.Println("Connecting to database...")
	connectionString := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true",
		env.Get("MYSQL_USER", ""),
		env.Get("MYSQL_PASSWORD", ""),
		env.Get("MYSQL_HOST", ""),
		env.Get("MYSQL_PORT", "3306"),
		env.Get("MYSQL_DATABASE", ""),
	)

	var err error
	connection, err = sqlx.Open("mysql", connectionString)

	if err != nil {
		panic(err.Error())
	}

	connection.SetConnMaxLifetime(time.Second * 10)
	connection.SetMaxOpenConns(50)
	connection.SetMaxIdleConns(50)

	return connection
}

func GetConnection() *sqlx.DB {
	return connection.Unsafe()
}

// HasConnection reports whether InitConnection has run - callers on hot paths
// that are also exercised by unit tests without a database (e.g. the CWMP
// conversation logger) should check this before calling GetConnection, which
// panics on a nil underlying connection.
func HasConnection() bool {
	return connection != nil
}
