package db

import (
	"database/sql"

	"fmt"

	"log"

	_ "github.com/go-sql-driver/mysql"

	"buildberry/internal/config"
)

var DB *sql.DB

func Connect(cfg config.Config) (*sql.DB, error) {

	dsn := fmt.Sprintf(

		"%s:%s@tcp(%s:%s)/%s",

		cfg.DBUSER,
		cfg.DBPASSWORD,
		cfg.DBHOST,
		cfg.DBPORT,
		cfg.DBNAME,
	)
	log.Println("database connect")

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	DB = db

	err = db.Ping()
	if err != nil {
		return nil, err

	}
	return db, nil

}
