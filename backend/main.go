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
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// Configuration
type Config struct {
	Port           string
	APIKey         string
	MongoURI       string
	DatabaseName   string
	AllowedOrigins []string
}

// User represents a user account
type User struct {
	ID        string    `json:"id" bson:"_id"`
	Email     string    `json:"email" bson:"email"`
	Password  string    `json:"-" bson:"password"`
	APIKey    string    `json:"api_key" bson:"api_key"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

// JSONDocument represents a stored JSON document
type JSONDocument struct {
	ID        string                 `json:"id" bson:"_id"`
	UserID    string                 `json:"user_id" bson:"user_id"`
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
	config          Config
	docCollection   *mongo.Collection
	usersCollection *mongo.Collection
	ctx             = context.Background()
)

func init() {
	// Load .env file
	godotenv.Load()

	config = Config{
		Port:           getEnv("PORT", "8080"),
		APIKey:         getEnv("API_KEY", ""),
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

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Println("Connected to MongoDB")

	db := client.Database(config.DatabaseName)
	docCollection = db.Collection("documents")
	usersCollection = db.Collection("users")

	// Create indexes
	docCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}},
	})
	usersCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	usersCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "api_key", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	// Setup routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", healthHandler)

	// Auth routes
	mux.HandleFunc("/auth/register", registerHandler)
	mux.HandleFunc("/auth/login", loginHandler)

	// API routes (protected)
	mux.HandleFunc("/api/documents", authMiddleware(documentsHandler))
	mux.HandleFunc("/api/documents/", authMiddleware(documentHandler))
	mux.HandleFunc("/api/me", authMiddleware(meHandler))

	// Public read endpoint
	mux.HandleFunc("/public/", publicHandler)

	handler := corsMiddleware(mux)

	addr := fmt.Sprintf(":%s", config.Port)
	log.Printf("JSON API Server starting on port %s", config.Port)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
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

// Auth middleware - supports both API key and legacy global API key
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		if apiKey == "" {
			sendJSON(w, http.StatusUnauthorized, APIResponse{
				Success: false,
				Error:   "API key is required",
			})
			return
		}

		// Check if it's the global API key (legacy support)
		if config.APIKey != "" && apiKey == config.APIKey {
			// Use global context
			r = r.WithContext(context.WithValue(r.Context(), "user_id", "global"))
			next(w, r)
			return
		}

		// Check user API key
		var user User
		err := usersCollection.FindOne(ctx, bson.M{"api_key": apiKey}).Decode(&user)
		if err != nil {
			sendJSON(w, http.StatusUnauthorized, APIResponse{
				Success: false,
				Error:   "Invalid API key",
			})
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), "user_id", user.ID))
		r = r.WithContext(context.WithValue(r.Context(), "user", user))
		next(w, r)
	}
}

func getUserID(r *http.Request) string {
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return userID
	}
	return ""
}

// Health handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "JSON API Server is running",
		Data: map[string]interface{}{
			"version":   "1.1.0",
			"storage":   "mongodb",
			"auth":      "email",
			"timestamp": time.Now().UTC(),
		},
	})
}

// Register handler
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &input); err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid JSON"})
		return
	}

	if input.Email == "" || input.Password == "" {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Email and password are required"})
		return
	}

	if len(input.Password) < 6 {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Password must be at least 6 characters"})
		return
	}

	// Check if email exists
	var existing User
	err := usersCollection.FindOne(ctx, bson.M{"email": strings.ToLower(input.Email)}).Decode(&existing)
	if err == nil {
		sendJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "Email already registered"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to create account"})
		return
	}

	// Create user
	user := User{
		ID:        uuid.New().String(),
		Email:     strings.ToLower(input.Email),
		Password:  string(hashedPassword),
		APIKey:    uuid.New().String(),
		CreatedAt: time.Now().UTC(),
	}

	_, err = usersCollection.InsertOne(ctx, user)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to create account"})
		return
	}

	sendJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "Account created successfully",
		Data: map[string]interface{}{
			"id":      user.ID,
			"email":   user.Email,
			"api_key": user.APIKey,
		},
	})
}

// Login handler
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &input); err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid JSON"})
		return
	}

	if input.Email == "" || input.Password == "" {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Email and password are required"})
		return
	}

	// Find user
	var user User
	err := usersCollection.FindOne(ctx, bson.M{"email": strings.ToLower(input.Email)}).Decode(&user)
	if err != nil {
		sendJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Error: "Invalid email or password"})
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		sendJSON(w, http.StatusUnauthorized, APIResponse{Success: false, Error: "Invalid email or password"})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"id":      user.ID,
			"email":   user.Email,
			"api_key": user.APIKey,
		},
	})
}

// Me handler - get current user info
func meHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	user, ok := r.Context().Value("user").(User)
	if !ok {
		sendJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"id":   "global",
				"type": "api_key",
			},
		})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"id":      user.ID,
			"email":   user.Email,
			"api_key": user.APIKey,
		},
	})
}

// Documents handler
func documentsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listDocuments(w, r)
	case http.MethodPost:
		createDocument(w, r)
	default:
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
	}
}

// Document handler
func documentHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/documents/")
	id := strings.TrimSuffix(path, "/")

	if id == "" {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Document ID is required"})
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
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
	}
}

// Public handler
func publicHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Only GET allowed"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/public/")
	id := strings.TrimSuffix(path, "/")

	if id == "" {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Document ID is required"})
		return
	}

	var doc JSONDocument
	err := docCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		sendJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "Document not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	json.NewEncoder(w).Encode(doc.Data)
}

// List documents for current user
func listDocuments(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	filter := bson.M{}
	if userID != "global" {
		filter["user_id"] = userID
	}

	cursor, err := docCollection.Find(ctx, filter)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to list documents"})
		return
	}
	defer cursor.Close(ctx)

	var docs []JSONDocument
	if err := cursor.All(ctx, &docs); err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to decode documents"})
		return
	}

	if docs == nil {
		docs = []JSONDocument{}
	}

	sendJSON(w, http.StatusOK, APIResponse{Success: true, Data: docs})
}

// Create document
func createDocument(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var input struct {
		Name string                 `json:"name"`
		Data map[string]interface{} `json:"data"`
	}

	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &input); err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid JSON"})
		return
	}

	if input.Name == "" {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Document name is required"})
		return
	}

	if input.Data == nil {
		input.Data = make(map[string]interface{})
	}

	doc := JSONDocument{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      input.Name,
		Data:      input.Data,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err := docCollection.InsertOne(ctx, doc)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to save document"})
		return
	}

	sendJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "Document created successfully",
		Data:    doc,
	})
}

// Get document
func getDocument(w http.ResponseWriter, r *http.Request, id string) {
	userID := getUserID(r)

	filter := bson.M{"_id": id}
	if userID != "global" {
		filter["user_id"] = userID
	}

	var doc JSONDocument
	err := docCollection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		sendJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "Document not found"})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{Success: true, Data: doc})
}

// Update document
func updateDocument(w http.ResponseWriter, r *http.Request, id string) {
	userID := getUserID(r)

	filter := bson.M{"_id": id}
	if userID != "global" {
		filter["user_id"] = userID
	}

	var existingDoc JSONDocument
	err := docCollection.FindOne(ctx, filter).Decode(&existingDoc)
	if err != nil {
		sendJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "Document not found"})
		return
	}

	var input struct {
		Name string                 `json:"name"`
		Data map[string]interface{} `json:"data"`
	}

	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &input); err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid JSON"})
		return
	}

	update := bson.M{"$set": bson.M{"updated_at": time.Now().UTC()}}
	if input.Name != "" {
		update["$set"].(bson.M)["name"] = input.Name
		existingDoc.Name = input.Name
	}
	if input.Data != nil {
		update["$set"].(bson.M)["data"] = input.Data
		existingDoc.Data = input.Data
	}

	_, err = docCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to update"})
		return
	}

	existingDoc.UpdatedAt = time.Now().UTC()
	sendJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Document updated", Data: existingDoc})
}

// Delete document
func deleteDocument(w http.ResponseWriter, r *http.Request, id string) {
	userID := getUserID(r)

	filter := bson.M{"_id": id}
	if userID != "global" {
		filter["user_id"] = userID
	}

	result, err := docCollection.DeleteOne(ctx, filter)
	if err != nil || result.DeletedCount == 0 {
		sendJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "Document not found"})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Document deleted"})
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
