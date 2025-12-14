# JSON API - Simple JSON Storage API

A lightweight Go API for storing and serving JSON data, with a Next.js dashboard for management.

## Project Structure

```
json-api/
├── backend/          # Go API (Deploy to Render)
│   ├── main.go
│   ├── go.mod
│   ├── Dockerfile
│   └── render.yaml
└── frontend/         # Next.js Dashboard (Deploy to Vercel)
    └── src/app/
        └── page.tsx
```

## Features

- RESTful API for JSON document CRUD operations
- API Key authentication
- Public read-only endpoints for websites
- Clean, Supabase-inspired dashboard

## API Endpoints

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

1. Create a new **Web Service** on Render
2. Connect your GitHub repository
3. Configure:
   - **Root Directory**: `backend`
   - **Runtime**: Docker
4. Add environment variable:
   - `API_KEY`: Your secret API key
5. Add a **Disk**:
   - **Mount Path**: `/app/data`
   - **Size**: 1 GB

### Frontend (Vercel)

1. Import your GitHub repository on Vercel
2. Configure:
   - **Root Directory**: `frontend`
   - **Framework Preset**: Next.js
3. Add environment variable (optional):
   - `NEXT_PUBLIC_API_URL`: Your Render backend URL

## Local Development

### Backend
```bash
cd backend
go mod tidy
API_KEY=your-secret-key go run main.go
# Server runs on http://localhost:8080
```

### Frontend
```bash
cd frontend
npm install
npm run dev
# Dashboard runs on http://localhost:3000
```

## Using in Your Website

```javascript
// Fetch data from public endpoint (no auth needed)
const response = await fetch('https://your-api.onrender.com/public/DOCUMENT_ID');
const data = await response.json();
```

## License

MIT
