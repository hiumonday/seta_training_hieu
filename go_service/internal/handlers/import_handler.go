package handlers

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"go_service/internal/services"

	"github.com/gin-gonic/gin"
)

type ImportHandler struct {
	userService *services.UserService
}

func NewImportHandler() *ImportHandler {
	return &ImportHandler{
		userService: services.NewUserService(),
	}
}

// Result represents the result of a user import operation
type ImportResult struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// ImportUsers handles the POST /import-users endpoint
func (h *ImportHandler) ImportUsers(c *gin.Context) {
	// Get the uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Printf("Error getting file from request: %v", err)

		// Check if the form was submitted correctly
		if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Failed to parse form data: " + err.Error(),
			})
			return
		}

		// Log more details about the request to help debug
		log.Printf("Form values: %v", c.Request.Form)
		log.Printf("File values: %v", c.Request.MultipartForm.File)

		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No file was uploaded or invalid file field. Please use 'file' as the form field name.",
			"details": err.Error(),
		})
		return
	}
	defer file.Close()

	// Log file details for debugging
	log.Printf("Received file: %s, size: %d bytes, content type: %s",
		header.Filename, header.Size, header.Header.Get("Content-Type"))

	// Check if the file is a CSV
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") &&
		header.Header.Get("Content-Type") != "text/csv" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Uploaded file must be a CSV file",
		})
		return
	}

	// Parse the CSV file
	reader := csv.NewReader(bufio.NewReader(file))
	reader.TrimLeadingSpace = true

	// Read header to validate format
	csvHeaders, err := reader.Read()
	if err != nil {
		log.Printf("Error reading CSV header: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid CSV format: could not read header",
		})
		return
	}

	// Check if the header contains the required fields
	expectedHeaders := []string{"username", "email", "password", "role"}
	if !validateHeaders(csvHeaders, expectedHeaders) {
		log.Printf("Invalid CSV headers: %v", csvHeaders)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid CSV format: header must contain username, email, password, role",
			"found":   csvHeaders,
		})
		return
	}

	// Create a buffered channel for the worker pool
	const maxWorkers = 5
	jobs := make(chan []string, 100)
	results := make(chan ImportResult, 100)
	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go h.worker(jobs, results, &wg)
	}

	// Producer: Read CSV rows and send to workers
	rowCount := 0
	go func() {
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Error reading CSV row %d: %v", rowCount+1, err)
				continue
			}
			rowCount++
			jobs <- record
		}
		log.Printf("Finished reading CSV file, found %d data rows", rowCount)
		close(jobs)
	}()

	// Collect results in a separate goroutine
	var allResults []ImportResult
	var successCount, failCount int
	resultsDone := make(chan bool)

	go func() {
		for result := range results {
			allResults = append(allResults, result)
			if result.Success {
				successCount++
			} else {
				failCount++
			}
		}
		resultsDone <- true
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(results)
	<-resultsDone

	// Return summary
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Imported %d users, %d succeeded, %d failed", len(allResults), successCount, failCount),
		"data": gin.H{
			"total":     len(allResults),
			"success":   successCount,
			"failed":    failCount,
			"results":   allResults,
			"totalRows": rowCount,
			"fileName":  header.Filename,
		},
	})
}

// worker processes user creation jobs
func (h *ImportHandler) worker(jobs <-chan []string, results chan<- ImportResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for record := range jobs {
		if len(record) < 4 {
			results <- ImportResult{
				Success: false,
				Error:   "Invalid record format: insufficient fields",
			}
			continue
		}

		username := strings.TrimSpace(record[0])
		email := strings.TrimSpace(record[1])
		password := strings.TrimSpace(record[2])
		role := strings.ToUpper(strings.TrimSpace(record[3]))

		// Validate fields
		if username == "" || email == "" || password == "" || role == "" {
			results <- ImportResult{
				Username: username,
				Email:    email,
				Success:  false,
				Error:    "Missing required fields",
			}
			continue
		}

		// Call user service to create user
		resp, err := h.userService.CreateUser(username, email, password, role)
		if err != nil {
			results <- ImportResult{
				Username: username,
				Email:    email,
				Success:  false,
				Error:    err.Error(),
			}
			continue
		}

		results <- ImportResult{
			Username: username,
			Email:    email,
			Success:  true,
		}

		log.Printf("Created user: %s (%s) with ID: %s", username, email, resp.User.ID)
	}
}

// validateHeaders checks if all expected headers are present in the actual headers
// validateHeaders checks if all expected headers are present in the actual headers
func validateHeaders(actual []string, expected []string) bool {
	if len(actual) < len(expected) {
		log.Printf("Header count mismatch: expected %d, got %d", len(expected), len(actual))
		return false
	}

	// Convert actual headers to lowercase for case-insensitive comparison
	// and remove BOM characters if present
	lowerActual := make([]string, len(actual))
	for i, h := range actual {
		// Remove BOM (Byte Order Mark) if present
		h = strings.TrimPrefix(h, "\ufeff")
		// Convert to lowercase and trim spaces
		lowerActual[i] = strings.ToLower(strings.TrimSpace(h))
		log.Printf("Header %d: original='%s', cleaned='%s'", i, actual[i], lowerActual[i])
	}

	for _, expectedHeader := range expected {
		expectedLower := strings.ToLower(expectedHeader)
		found := false
		for _, actualHeader := range lowerActual {
			if expectedLower == actualHeader {
				found = true
				break
			}
		}
		if !found {
			log.Printf("Missing expected header: '%s'", expectedHeader)
			return false
		}
	}

	return true
}
