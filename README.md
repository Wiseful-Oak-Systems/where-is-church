# Where is Church?

A way to quickly identify the churches close to us, so we don't need to stress about missing mass.

## Features

- **Interactive Map** - Find nearby churches on an OpenStreetMap-powered map with Leaflet.js
- **Geospatial Search** - Search by radius (1-50km) with Haversine distance calculation
- **Denomination Filter** - Default Catholic, with support for Orthodox, Protestant, Anglican, Evangelical
- **User Authentication** - JWT-based auth with email/password registration
- **Role System** - User, Moderator, and Admin roles with escalating permissions
- **Check-ins** - Foursquare-style "I was here!" check-ins to track church attendance
- **Loyalty Tracking** - Most visited churches, attendance stats, and loyal user identification
- **Mass Schedules** - Weekly mass times with day, time, language, and notes
- **Suggestions** - Users can suggest new churches, corrections, and schedule updates
- **Admin Dashboard** - Manage users, verify churches, and review suggestions
- **Responsive Design** - Mobile-first with sidebar, bottom sheet, and touch-friendly UI

## Tech Stack

- **Backend**: Go (Gin framework)
- **Database**: PostgreSQL with PostGIS extension
- **Auth**: JWT tokens with bcrypt password hashing
- **ORM**: GORM with auto-migrations
- **Frontend**: Vanilla JS, Leaflet.js (OpenStreetMap), responsive CSS
- **Maps**: OpenStreetMap tiles via Leaflet.js

## Getting Started

### Prerequisites

- Go 1.22+
- Docker & Docker Compose (for PostgreSQL)

### Setup

```bash
# Start PostgreSQL with PostGIS
make db-up

# Copy and configure environment variables
cp .env.example .env

# Run the server
make run
```

The app will be available at `http://localhost:8080`.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | Database host |
| `DB_PORT` | `5432` | Database port |
| `DB_USER` | `wic` | Database user |
| `DB_PASSWORD` | `wic_dev_password` | Database password |
| `DB_NAME` | `whereischurch` | Database name |
| `JWT_SECRET` | `change-me-to-a-random-secret` | JWT signing secret |
| `JWT_EXPIRY_HOURS` | `72` | Token expiry in hours |
| `PORT` | `8080` | Server port |

## API Endpoints

### Auth
- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - Login
- `POST /api/auth/logout` - Logout
- `GET /api/auth/me` - Get current user
- `PUT /api/auth/profile` - Update profile

### Churches
- `GET /api/churches` - List all churches
- `GET /api/churches/search?lat=X&lng=Y&radius=Z&denomination=D` - Search nearby
- `GET /api/churches/:id` - Get church details
- `POST /api/churches` - Create church
- `PUT /api/churches/:id` - Update church (mod/admin)
- `POST /api/churches/:id/schedules` - Add mass schedule (mod/admin)

### Check-ins
- `POST /api/checkins` - Check in at a church
- `GET /api/checkins/mine` - My check-in history
- `GET /api/checkins/stats` - My attendance stats
- `GET /api/churches/:id/checkins` - Church check-in history
- `GET /api/churches/:id/loyal-users` - Most loyal visitors

### Suggestions
- `POST /api/suggestions` - Submit a suggestion
- `GET /api/suggestions/mine` - My suggestions
- `GET /api/suggestions` - List suggestions (mod/admin)
- `PUT /api/suggestions/:id` - Review suggestion (mod/admin)

### Admin
- `GET /api/admin/users` - List all users
- `PUT /api/admin/users/:id/role` - Change user role
- `PUT /api/admin/churches/:id/verify` - Verify a church
- `DELETE /api/admin/churches/:id` - Delete a church

## User Roles

| Role | Permissions |
|------|-------------|
| **User** | Search, check-in, suggest |
| **Moderator** | + Edit churches, manage schedules, review suggestions |
| **Admin** | + Manage users, verify/delete churches, set roles |

Moderators are promoted from loyal users who regularly check in at churches.
