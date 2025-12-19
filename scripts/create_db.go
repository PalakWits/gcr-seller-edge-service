package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// Connect to postgres database to create new database
	db, err := sql.Open("postgres", "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check if database exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = 'user')").Scan(&exists)
	if err != nil {
		log.Fatal(err)
	}

	if !exists {
		// Terminate existing connections to template1
		_, _ = db.Exec("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'template1' AND pid <> pg_backend_pid()")

		// Create database without template
		_, err = db.Exec(`CREATE DATABASE "user" WITH TEMPLATE template0`)
		if err != nil {
			// If it still fails, just use the simpler command
			_, err = db.Exec(`CREATE DATABASE "user"`)
			if err != nil {
				log.Printf("Warning: Could not create database: %v", err)
				log.Println("You may need to create it manually using: CREATE DATABASE \"user\";")
			} else {
				fmt.Println("Database 'user' created successfully!")
			}
		} else {
			fmt.Println("Database 'user' created successfully!")
		}
	} else {
		fmt.Println("Database 'user' already exists.")
	}
}
