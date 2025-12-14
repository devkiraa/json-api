# JSON API - Simple JSON Storage API

A lightweight Go API for storing and serving JSON data with MongoDB, plus a Next.js dashboard for management.

## Live Deployments

- **Dashboard**: https://json-api-lac.vercel.app
- **API**: https://json-api-5wyk.onrender.com

## Features

- RESTful API for JSON document CRUD operations
- **MongoDB** storage for reliable data persistence
- API Key authentication
- Public read-only endpoints for websites
- Clean, Supabase-inspired dashboard

## Project Structure

```
json-api/
├── backend/          # Go API (Render)
│   └── main.go       # MongoDB-backed API server
└── frontend/         # Next.js Dashboard (Vercel)
    └── src/app/
```

## Environment Variables

### Backend (Render)

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | No | Server port (default: 8080) |
| `API_KEY` | Yes | Your secret API key |
| `MONGODB_URI` | Yes | MongoDB connection string |
| `DATABASE_NAME` | No | Database name (default: jsonapi) |
| `ALLOWED_ORIGINS` | No | CORS origins (default: *) |

## API Endpoints

Base URL: `https://json-api-5wyk.onrender.com`

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/health` | No | Health check |
| GET | `/api/documents` | Yes | List all documents |
| POST | `/api/documents` | Yes | Create document |
| GET | `/api/documents/:id` | Yes | Get document |
| PUT | `/api/documents/:id` | Yes | Update document |
| DELETE | `/api/documents/:id` | Yes | Delete document |
| GET | `/public/:id` | No | Public read access |

## Deployment

### Backend (Render.com)

1. Create a new **Web Service**
2. Connect your GitHub repository
3. Configure:
   - **Root Directory**: `backend`
   - **Runtime**: Docker
4. Add environment variables:
   - `API_KEY`: Your secret API key
   - `MONGODB_URI`: Your MongoDB connection string (e.g., from MongoDB Atlas)
   - `DATABASE_NAME`: jsonapi (or your preferred name)

### Frontend (Vercel)

1. Import your GitHub repository
2. Configure:
   - **Root Directory**: `frontend`
   - **Framework Preset**: Next.js

## MongoDB Setup

### Option 1: MongoDB Atlas (Recommended for Production)

1. Go to [MongoDB Atlas](https://www.mongodb.com/atlas)
2. Create a free cluster
3. Create a database user
4. Get your connection string
5. Add to Render environment: `MONGODB_URI=mongodb+srv://user:pass@cluster.mongodb.net/`

### Option 2: Local MongoDB

```bash
# Start MongoDB locally
mongod --dbpath /data/db

# Set environment variable
export MONGODB_URI=mongodb://localhost:27017
```

## Local Development

### Backend
```bash
cd backend
go mod tidy
MONGODB_URI=mongodb://localhost:27017 API_KEY=your-key go run main.go
```

### Frontend
```bash
cd frontend
npm install
npm run dev
```

## Using in Your Website

```javascript
// Fetch data (no auth needed for public endpoints)
const response = await fetch('https://json-api-5wyk.onrender.com/public/YOUR_DOC_ID');
const data = await response.json();
```

## License

MIT
