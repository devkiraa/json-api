package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Configuration
type Config struct {
	Port       string
	APIKey     string
	DataDir    string
	AllowedOrigins []string
}

// JSONStore handles all JSON data operations
type JSONStore struct {
	mu      sync.RWMutex
	dataDir string
}

// JSONDocument represents a stored JSON document
type JSONDocument struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// APIResponse is a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

var (
	config Config
	store  *JSONStore
)

func init() {
	// Load configuration from environment variables
	config = Config{
		Port:       getEnv("PORT", "8080"),
		APIKey:     getEnv("API_KEY", "your-secret-api-key-change-me"),
		DataDir:    getEnv("DATA_DIR", "./data"),
		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "*"), ","),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize the JSON store
	store = NewJSONStore(config.DataDir)

	// Setup routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", healthHandler)

	// API routes (protected)
	mux.HandleFunc("/api/documents", authMiddleware(documentsHandler))
	mux.HandleFunc("/api/documents/", authMiddleware(documentHandler))

	// Public read endpoint (for websites to consume)
	mux.HandleFunc("/public/", publicHandler)

	// Wrap with CORS middleware
	handler := corsMiddleware(mux)

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	log.Printf("🚀 JSON API Server starting on port %s", config.Port)
	log.Printf("📁 Data directory: %s", config.DataDir)
	log.Printf("🔑 API Key configured: %s***", config.APIKey[:min(8, len(config.APIKey))])

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NewJSONStore creates a new JSON store
func NewJSONStore(dataDir string) *JSONStore {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	return &JSONStore{
		dataDir: dataDir,
	}
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range config.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if len(config.AllowedOrigins) > 0 && config.AllowedOrigins[0] == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Auth middleware
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		if apiKey != config.APIKey {
			sendJSON(w, http.StatusUnauthorized, APIResponse{
				Success: false,
				Error:   "Invalid or missing API key",
			})
			return
		}

		next(w, r)
	}
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "JSON API Server is running",
		Data: map[string]interface{}{
			"version":   "1.0.0",
			"timestamp": time.Now().UTC(),
		},
	})
}

// Documents handler (list all, create new)
func documentsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listDocuments(w, r)
	case http.MethodPost:
		createDocument(w, r)
	default:
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
	}
}

// Document handler (get, update, delete single document)
func documentHandler(w http.ResponseWriter, r *http.Request) {
	// Extract document ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/documents/")
	id := strings.TrimSuffix(path, "/")

	if id == "" {
		sendJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Document ID is required",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		getDocument(w, r, id)
	case http.MethodPut:
		updateDocument(w, r, id)
	case http.MethodDelete:
		deleteDocument(w, r, id)
	default:
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
	}
}

// Public handler for read-only access
func publicHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Error:   "Only GET method is allowed for public access",
		})
		return
	}

	// Extract document ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/public/")
	id := strings.TrimSuffix(path, "/")

	if id == "" {
		sendJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Document ID is required",
		})
		return
	}

	doc, err := store.Get(id)
	if err != nil {
		sendJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Document not found",
		})
		return
	}

	// Return just the data for public access
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	w.Write(doc.Data)
}

// List all documents
func listDocuments(w http.ResponseWriter, r *http.Request) {
	docs, err := store.List()
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to list documents: %v", err),
		})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    docs,
	})
}

// Create a new document
func createDocument(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Failed to read request body",
		})
		return
	}
	defer r.Body.Close()

	var input struct {
		Name string          `json:"name"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(body, &input); err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON format",
		})
		return
	}

	if input.Name == "" {
		sendJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Document name is required",
		})
		return
	}

	if len(input.Data) == 0 {
		input.Data = json.RawMessage("{}")
	}

	doc := &JSONDocument{
		ID:        uuid.New().String(),
		Name:      input.Name,
		Data:      input.Data,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := store.Save(doc); err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to save document: %v", err),
		})
		return
	}

	sendJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "Document created successfully",
		Data:    doc,
	})
}

// Get a single document
func getDocument(w http.ResponseWriter, r *http.Request, id string) {
	doc, err := store.Get(id)
	if err != nil {
		sendJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Document not found",
		})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    doc,
	})
}

// Update a document
func updateDocument(w http.ResponseWriter, r *http.Request, id string) {
	doc, err := store.Get(id)
	if err != nil {
		sendJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Document not found",
		})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Failed to read request body",
		})
		return
	}
	defer r.Body.Close()

	var input struct {
		Name string          `json:"name"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(body, &input); err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON format",
		})
		return
	}

	if input.Name != "" {
		doc.Name = input.Name
	}
	if len(input.Data) > 0 {
		doc.Data = input.Data
	}
	doc.UpdatedAt = time.Now().UTC()

	if err := store.Save(doc); err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to update document: %v", err),
		})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Document updated successfully",
		Data:    doc,
	})
}

// Delete a document
func deleteDocument(w http.ResponseWriter, r *http.Request, id string) {
	if err := store.Delete(id); err != nil {
		sendJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Document not found",
		})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Document deleted successfully",
	})
}

// Store methods
func (s *JSONStore) Save(doc *JSONDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := filepath.Join(s.dataDir, doc.ID+".json")
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func (s *JSONStore) Get(id string) (*JSONDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filename := filepath.Join(s.dataDir, id+".json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var doc JSONDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

func (s *JSONStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := filepath.Join(s.dataDir, id+".json")
	return os.Remove(filename)
}

func (s *JSONStore) List() ([]*JSONDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, err
	}

	var docs []*JSONDocument
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		id := strings.TrimSuffix(file.Name(), ".json")
		doc, err := s.Get(id)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// Helper function to send JSON response
func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
