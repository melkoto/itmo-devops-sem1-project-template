package repository

import (
	"archive/zip"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

type PriceRepository interface {
	ImportPrices(reader *csv.Reader) (map[string]interface{}, error)
	ExportPrices() (string, error)
	Close() error
}

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo() (*PostgresRepo, error) {
	connectionString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("PG_HOST"),
		os.Getenv("PG_PORT"),
		os.Getenv("PG_USER"),
		os.Getenv("PG_PASSWORD"),
		os.Getenv("PG_DBNAME"),
		os.Getenv("PG_SSLMODE"),
	)
	database, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}

	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	return &PostgresRepo{db: database}, nil
}

func (repository *PostgresRepo) ImportPrices(csvReader *csv.Reader) (map[string]interface{}, error) {
	transaction, err := repository.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer transaction.Rollback()

	statement, err := transaction.Prepare(
		`INSERT INTO prices(id, product_name, category, price, created_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare SQL statement: %w", err)
	}
	defer statement.Close()

	statistics := map[string]interface{}{
		"total_items":      0,
		"total_categories": make(map[string]struct{}),
		"total_price":      0.0,
	}

	for {
		record, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading CSV: %w", err)
		}

		if len(record) != 5 {
			return nil, fmt.Errorf("invalid record length: %d, expected 5", len(record))
		}

		id := record[0]
		productName := record[1]
		category := record[2]
		price, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid price format in record: %w", err)
		}
		createdAt, err := time.Parse("2006-01-02", record[4])
		if err != nil {
			return nil, fmt.Errorf("invalid date format in record: %w", err)
		}

		_, err = statement.Exec(id, productName, category, price, createdAt)
		if err != nil {
			return nil, fmt.Errorf("error inserting data into database: %w", err)
		}

		statistics["total_items"] = statistics["total_items"].(int) + 1
		statistics["total_categories"].(map[string]struct{})[category] = struct{}{}
		statistics["total_price"] = statistics["total_price"].(float64) + price
	}

	if err := transaction.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return map[string]interface{}{
		"total_items":      statistics["total_items"],
		"total_categories": len(statistics["total_categories"].(map[string]struct{})),
		"total_price":      statistics["total_price"],
	}, nil
}

func (repository *PostgresRepo) ExportPrices() (string, error) {
	rows, err := repository.db.Query("SELECT id, created_at, product_name, category, price FROM prices")
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	tempFile, err := os.CreateTemp("", "export-*.csv")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	csvWriter := csv.NewWriter(tempFile)
	defer csvWriter.Flush()

	if err := csvWriter.Write([]string{"id", "created_at", "product_name", "category", "price"}); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	for rows.Next() {
		var (
			id          string
			createdAt   time.Time
			productName string
			category    string
			price       float64
		)

		if err := rows.Scan(&id, &createdAt, &productName, &category, &price); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}

		record := []string{
			id,
			createdAt.Format("2006-01-02"),
			productName,
			category,
			strconv.FormatFloat(price, 'f', 2, 64),
		}

		if err := csvWriter.Write(record); err != nil {
			return "", fmt.Errorf("failed to write record to CSV: %w", err)
		}
	}

	csvWriter.Flush()

	zipFile, err := os.CreateTemp("", "export-*.zip")
	if err != nil {
		return "", fmt.Errorf("failed to create ZIP file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	dataFile, err := zipWriter.Create("data.csv")
	if err != nil {
		return "", fmt.Errorf("failed to create ZIP entry: %w", err)
	}

	sourceFile, err := os.Open(tempFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer sourceFile.Close()

	if _, err := io.Copy(dataFile, sourceFile); err != nil {
		return "", fmt.Errorf("failed to copy CSV data to ZIP: %w", err)
	}

	return zipFile.Name(), nil
}

func (repository *PostgresRepo) Close() error {
	return repository.db.Close()
}
