package repo

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func InitDB(cfg *Config) (*sql.DB, error) {
	dbCfg := cfg.Database

	// Connect without database to create it if needed.
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port)

	var db *sql.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping()
		}

		if err == nil {
			break
		}

		log.Printf("Failed to connect to database (attempt %d/10): %v", i+1, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect after 10 attempts: %w", err)
	}

	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + dbCfg.Name)

	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	db.Close()

	// Reconnect using the target database.
	dsnWithDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.Name)

	db, err = sql.Open("mysql", dsnWithDB)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database %s: %w", dbCfg.Name, err)
	}

	return db, nil
}
