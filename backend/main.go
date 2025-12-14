package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Configuration
type Config struct {
	Port           string
	APIKey         string
	MongoURI       string
	DatabaseName   string
	AllowedOrigins []string
}

// JSONDocument represents a stored JSON document
type JSONDocument struct {
	ID        string                 `json:"id" bson:"_id"`
	Name      string                 `json:"name" bson:"name"`
	Data      map[string]interface{} `json:"data" bson:"data"`
	CreatedAt time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" bson:"updated_at"`
}

// APIResponse is a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

var (
	config     Config
	collection *mongo.Collection
	ctx        = context.Background()
)

func init() {
	config = Config{
		Port:           getEnv("PORT", "8080"),
		APIKey:         getEnv("API_KEY", "your-secret-api-key-change-me"),
		MongoURI:       getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DatabaseName:   getEnv("DATABASE_NAME", "jsonapi"),
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
	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(config.MongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Ping MongoDB
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Println("Connected to MongoDB")

	// Get collection
	collection = client.Database(config.DatabaseName).Collection("documents")

	// Create index on name field
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "name", Value: 1}},
	}
	collection.Indexes().CreateOne(ctx, indexModel)

	// Setup routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", healthHandler)

	// API routes (protected)
	mux.HandleFunc("/api/documents", authMiddleware(documentsHandler))
	mux.HandleFunc("/api/documents/", authMiddleware(documentHandler))

	// Public read endpoint
	mux.HandleFunc("/public/", publicHandler)

	// Wrap with CORS middleware
	handler := corsMiddleware(mux)

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	log.Printf("JSON API Server starting on port %s", config.Port)
	log.Printf("API Key configured: %s***", config.APIKey[:min(8, len(config.APIKey))])

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

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

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
			"storage":   "mongodb",
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

	path := strings.TrimPrefix(r.URL.Path, "/public/")
	id := strings.TrimSuffix(path, "/")

	if id == "" {
		sendJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Document ID is required",
		})
		return
	}

	var doc JSONDocument
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		sendJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Document not found",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(doc.Data)
}

// List all documents
func listDocuments(w http.ResponseWriter, r *http.Request) {
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to list documents: %v", err),
		})
		return
	}
	defer cursor.Close(ctx)

	var docs []JSONDocument
	if err := cursor.All(ctx, &docs); err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to decode documents: %v", err),
		})
		return
	}

	if docs == nil {
		docs = []JSONDocument{}
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
		Name string                 `json:"name"`
		Data map[string]interface{} `json:"data"`
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

	if input.Data == nil {
		input.Data = make(map[string]interface{})
	}

	doc := JSONDocument{
		ID:        uuid.New().String(),
		Name:      input.Name,
		Data:      input.Data,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err = collection.InsertOne(ctx, doc)
	if err != nil {
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
	var doc JSONDocument
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
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
	var existingDoc JSONDocument
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&existingDoc)
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
		Name string                 `json:"name"`
		Data map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(body, &input); err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON format",
		})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now().UTC(),
		},
	}

	if input.Name != "" {
		update["$set"].(bson.M)["name"] = input.Name
		existingDoc.Name = input.Name
	}
	if input.Data != nil {
		update["$set"].(bson.M)["data"] = input.Data
		existingDoc.Data = input.Data
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to update document: %v", err),
		})
		return
	}

	existingDoc.UpdatedAt = time.Now().UTC()

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Document updated successfully",
		Data:    existingDoc,
	})
}

// Delete a document
func deleteDocument(w http.ResponseWriter, r *http.Request, id string) {
	result, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil || result.DeletedCount == 0 {
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

// Helper function to send JSON response
func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
