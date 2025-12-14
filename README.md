# JSON API - Simple JSON Storage API with Dashboard

A lightweight Go API for storing and serving JSON data, with a beautiful modern dashboard for management. Perfect for static websites, prototypes, or any application that needs simple JSON storage.

## 🚀 Features

- **RESTful API** - CRUD operations for JSON documents
- **API Key Authentication** - Secure your data with a simple API key
- **Public Endpoints** - Share read-only access to your data
- **Modern Dashboard** - Beautiful dark-themed UI to manage your data
- **Render.com Ready** - Deploy in minutes with included configuration
- **Persistent Storage** - Data stored as JSON files with optional disk persistence

## 📁 Project Structure

```
json-api/
├── backend/
│   ├── main.go           # Go API server
│   ├── go.mod            # Go module definition
│   ├── Dockerfile        # Docker configuration
│   └── render.yaml       # Render.com deployment config
├── frontend/
│   ├── index.html        # Dashboard HTML
│   ├── styles.css        # Modern CSS styling
│   └── app.js           # Dashboard JavaScript
└── README.md
```

## 🛠️ Quick Start

### Running Locally

1. **Start the API Server:**
   ```bash
   cd backend
   go mod tidy
   API_KEY=your-secret-key go run main.go
   ```

2. **Open the Dashboard:**
   - Open `frontend/index.html` in your browser
   - Enter `http://localhost:8080` as the API URL
   - Enter your API key

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | Server port |
| `API_KEY` | your-secret-api-key-change-me | API authentication key |
| `DATA_DIR` | ./data | Directory for storing JSON files |
| `ALLOWED_ORIGINS` | * | CORS allowed origins |

## 🌐 Deploy to Render.com

### Backend Deployment

1. Push this repository to GitHub
2. Create a new **Web Service** on Render
3. Connect your repository
4. Configure settings:
   - **Root Directory**: `backend`
   - **Runtime**: Docker
5. Add environment variable:
   - `API_KEY`: Your secret API key (generate a strong one!)
6. Add a **Disk** for persistent storage:
   - **Mount Path**: `/app/data`
   - **Size**: 1 GB (or more as needed)

### Frontend Deployment

1. Create a new **Static Site** on Render
2. Connect the same repository
3. Configure settings:
   - **Root Directory**: `frontend`
   - **Build Command**: (leave empty)
   - **Publish Directory**: `.`

## 📡 API Reference

### Authentication

All `/api/*` endpoints require the `X-API-Key` header:
```
X-API-Key: your-api-key
```

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/documents` | List all documents |
| POST | `/api/documents` | Create a document |
| GET | `/api/documents/{id}` | Get a document |
| PUT | `/api/documents/{id}` | Update a document |
| DELETE | `/api/documents/{id}` | Delete a document |
| GET | `/public/{id}` | Public read-only access |

### Create Document

```bash
curl -X POST "https://your-api.onrender.com/api/documents" \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "products",
    "data": {
      "items": [
        {"id": 1, "name": "Widget", "price": 9.99}
      ]
    }
  }'
```

### Response Format

All API responses follow this format:
```json
{
  "success": true,
  "message": "Optional message",
  "data": { },
  "error": "Error message if success is false"
}
```

## 🌍 Using in Your Website

Once you have documents created, use the public endpoint in your website:

```html
<script>
  // Fetch your data (no API key needed for public access)
  fetch('https://your-api.onrender.com/public/YOUR_DOCUMENT_ID')
    .then(res => res.json())
    .then(data => {
      // Use your data
      console.log(data);
    });
</script>
```

## 🔒 Security Notes

1. **Generate a strong API key** - Use a random string of at least 32 characters
2. **Keep your API key secret** - Never expose it in frontend code
3. **Use HTTPS** - Render.com provides free SSL
4. **Set ALLOWED_ORIGINS** - Restrict to your domain in production

## 📝 License

MIT License - Feel free to use this project for any purpose.
