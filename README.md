# JSON API - Simple JSON Storage API

A lightweight Go API for storing and serving JSON data, with a Next.js dashboard for management.

## Live Deployments

- **Dashboard**: https://json-api-lac.vercel.app
- **API**: https://json-api-5wyk.onrender.com

## Project Structure

```
json-api/
├── backend/          # Go API (Render)
└── frontend/         # Next.js Dashboard (Vercel)
```

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

## Quick Start

1. Go to https://json-api-lac.vercel.app
2. Enter API URL: `https://json-api-5wyk.onrender.com`
3. Enter your API Key
4. Start managing your JSON documents!

## Using in Your Website

```javascript
// Fetch data (no auth needed for public endpoints)
const response = await fetch('https://json-api-5wyk.onrender.com/public/YOUR_DOC_ID');
const data = await response.json();
```

## Local Development

### Backend
```bash
cd backend
API_KEY=your-key go run main.go
```

### Frontend
```bash
cd frontend
npm run dev
```

## License

MIT
