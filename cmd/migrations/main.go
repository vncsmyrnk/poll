package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("a migration name is required.")
	}
	migrationName := os.Args[1]

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	connStr := dbConnString()
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	basePath := filepath.Join(".", "internal", "adapters", "repository", "postgres", "migrations")
	fileContent, err := migrationFileContent(basePath, migrationName)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(string(fileContent))
	if err != nil {
		log.Fatalf("Failed to execute SQL file: %v", err)
	}

	fmt.Println("Migration file executed successfully.")
}

func migrationFileContent(basePath string, migrationName string) ([]byte, error) {
	filePath, err := migrationFilePath(basePath, migrationName)
	if err != nil {
		return nil, err
	}

	fileContent, err := os.ReadFile(filepath.Join(basePath, filePath))
	if err != nil {
		return nil, err
	}

	return fileContent, nil
}

func migrationFilePath(basePath string, migrationName string) (string, error) {
	patternStr := fmt.Sprintf(`^.*%s\.sql`, regexp.QuoteMeta(migrationName))

	regex, err := regexp.Compile(patternStr)
	if err != nil {
		log.Fatalf("Invalid pattern: %v", err)
	}

	files, _ := os.ReadDir(basePath)
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if regex.MatchString(f.Name()) {
			return f.Name(), nil
		}
	}

	return "", fmt.Errorf("migration file not found")
}

func dbConnString() string {
	dbName, user, password, host, port := dbConfig()
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbName)
}

func dbConfig() (dbName string, user string, password string, host string, port string) {
	dbName = os.Getenv("POSTGRES_DB")
	user = os.Getenv("POSTGRES_USER")
	password = os.Getenv("POSTGRES_PASSWORD")
	host = os.Getenv("POSTGRES_HOST")
	port = os.Getenv("POSTGRES_PORT")
	return
}
