package handler

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"project_sem/internal/repository"
)

type PriceHandler struct {
	repo repository.PriceRepository
}

func NewPriceHandler(repo repository.PriceRepository) *PriceHandler {
	return &PriceHandler{repo: repo}
}

func (handler *PriceHandler) UploadPrices(writer http.ResponseWriter, request *http.Request) {
	file, _, err := request.FormFile("file")
	if err != nil {
		log.Printf("Failed to read uploaded file: %v", err)
		respondWithError(writer, http.StatusBadRequest, "Invalid file upload")
		return
	}
	defer file.Close()

	tmpFile, err := os.CreateTemp("", "upload-*.zip")
	if err != nil {
		log.Printf("Failed to create temporary file: %v", err)
		respondWithError(writer, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, file); err != nil {
		log.Printf("Failed to save uploaded file: %v", err)
		respondWithError(writer, http.StatusInternalServerError, "Failed to save file")
		return
	}

	stats, err := handler.processUpload(tmpFile.Name())
	if err != nil {
		log.Printf("Failed to process uploaded file: %v", err)
		respondWithError(writer, http.StatusBadRequest, err.Error())
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(stats)
}

func (handler *PriceHandler) DownloadPrices(writer http.ResponseWriter, request *http.Request) {
	zipPath, err := handler.repo.ExportPrices()
	if err != nil {
		respondWithError(writer, http.StatusInternalServerError, "Failed to export data")
		return
	}
	defer os.Remove(zipPath)

	writer.Header().Set("Content-Type", "application/zip")
	writer.Header().Set("Content-Disposition", "attachment; filename=data.zip")
	http.ServeFile(writer, request, zipPath)
}

func (handler *PriceHandler) processUpload(zipPath string) (map[string]interface{}, error) {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		log.Printf("Failed to open ZIP file: %v", err)
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}
	defer zipReader.Close()

	var csvFile *zip.File
	for _, file := range zipReader.File {
		if file.Name != "" && file.Name[len(file.Name)-4:] == ".csv" {
			csvFile = file
			break
		}
	}

	if csvFile == nil {
		return nil, errors.New("no CSV file found in archive")
	}

	rc, err := csvFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV: %w", err)
	}
	defer rc.Close()

	reader := csv.NewReader(rc)
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	stats, err := handler.repo.ImportPrices(reader)
	if err != nil {
		return nil, fmt.Errorf("import failed: %w", err)
	}

	return stats, nil
}

func respondWithError(writer http.ResponseWriter, code int, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	json.NewEncoder(writer).Encode(map[string]string{
		"error": message,
	})
}
