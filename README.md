# Where is Church?

[![CI](https://github.com/Wiseful-Oak-Systems/where-is-church/actions/workflows/ci.yml/badge.svg)](https://github.com/Wiseful-Oak-Systems/where-is-church/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/wiseful-oak-systems/where-is-church)](https://goreportcard.com/report/github.com/wiseful-oak-systems/where-is-church)
[![License](https://img.shields.io/github/license/Wiseful-Oak-Systems/where-is-church)](LICENSE)

Find churches near you. Never miss mass again.

An open-source, mobile-first church finder with Foursquare-style check-ins, community-driven data, and multi-denomination support. Built with Go, PostgreSQL/PostGIS, and Leaflet.js.

## Why This Exists

Existing church finder apps suffer from **stale data**, **poor mobile UX**, and **no engagement loop**. Catholic Mass Times has 130K+ churches but no way to verify if schedules are still correct. MassTimes.org hasn't been updated in years. ChurchFinder.com is a static directory.

**Where is Church?** solves this with:
- **Check-in loyalty** — Foursquare-style "I was here!" attendance tracking (no competitor has this)
- **Community moderation** — Active users earn moderator status, creating a self-sustaining data quality loop
- **Data freshness** — "Last verified" timestamps powered by real check-in activity
- **Multi-denomination** — Catholic, Orthodox, Protestant, Anglican, and Evangelical in one app

See [docs/MARKET_RESEARCH.md](docs/MARKET_RESEARCH.md) for our full competitive analysis.

## Features

### Core
- **Interactive Map** — OpenStreetMap via Leaflet.js with denomination-colored markers
- **Geospatial Search** — Haversine distance calculation with configurable radius (1-100km)
- **Denomination Filter** — Default based on user profile, with "All" option
- **Mass / Confession / Adoration Schedules** — Weekly times with day, time range, language, and notes
- **Favorites / Bookmarks** — Quick access to your regular churches

### Social
- **Check-ins** — "I was here!" with 2-hour duplicate prevention
- **Loyalty Tracking** — Attendance stats, top churches, most loyal users per parish
- **Suggestions** — Submit new churches, corrections, schedule updates

### Governance
- **Role System** — User > Moderator > Admin with escalating permissions
- **Moderator Promotion** — Admins promote loyal users who actively check in
- **Suggestion Review** — Structured approve/reject workflow with reviewer notes

### Technical
- **JWT Authentication** — Secure tokens with algorithm validation, issuer claims, SameSite cookies
- **Graceful Shutdown** — SIGTERM handling with request draining
- **Health Check** — `/health` endpoint with DB connectivity check
- **Responsive Design** — Mobile-first with dark mode support
- **PWA Ready** — Manifest for "Add to Home Screen" on mobile

## Quick Start

### Prerequisites
- Go 1.22+
- Docker & Docker Compose

### Run Locally
```bash
git clone https://github.com/Wiseful-Oak-Systems/where-is-church.git
cd where-is-church
cp .env.example .env
make dev    # starts PostgreSQL + the server
```

Open http://localhost:8080, register an account, and start finding churches.

### Run with Docker
```bash
docker compose up -d      # start PostgreSQL
make docker-build         # build the app image
docker run -p 8080:8080 --env-file .env where-is-church:latest
```

### Run Tests
```bash
make test            # run all tests
make test-verbose    # verbose output
make coverage        # generate coverage report
make lint            # run golangci-lint
```

## Project Structure

```
cmd/server/             # Application entrypoint with graceful shutdown
internal/
  config/               # Env-based configuration with production validation
  database/             # PostgreSQL/PostGIS connection and migrations
  handlers/             # HTTP handlers — auth, churches, check-ins, suggestions, admin, favorites
  middleware/            # JWT auth (algorithm-validated) and role-based access control
  models/               # Domain models — User, Church, MassSchedule, CheckIn, Suggestion, Favorite
  testutil/             # Shared test infrastructure (in-memory SQLite)
web/
  static/css/           # Responsive CSS with dark mode and skeleton loading
  static/js/            # Vanilla JS — Leaflet.js map, API client, UI interactions
  templates/            # Go HTML templates — index, church detail, profile, admin
docs/                   # Market research and architecture documentation
```

## API Reference

### Authentication
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/register` | Public | Register (name, email, password 8+, denomination) |
| POST | `/api/auth/login` | Public | Login, returns JWT |
| POST | `/api/auth/logout` | User | Clear session |
| GET | `/api/auth/me` | User | Current user profile |
| PUT | `/api/auth/profile` | User | Update name, denomination, location |

### Churches
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/churches` | User | List (paginated, denomination filter) |
| GET | `/api/churches/search?lat=&lng=&radius=` | User | Nearby search (Haversine) |
| GET | `/api/churches/:id` | User | Detail with schedules |
| POST | `/api/churches` | User | Create (auto-verified if mod/admin) |
| PUT | `/api/churches/:id` | Mod | Update details |
| POST | `/api/churches/:id/schedules` | Mod | Add mass/confession/adoration schedule |
| DELETE | `/api/churches/:id/schedules/:sid` | Mod | Remove schedule |

### Check-ins & Favorites
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/checkins` | User | Check in (2hr dedup) |
| GET | `/api/checkins/mine` | User | My history |
| GET | `/api/checkins/stats` | User | My stats (total, 30d, top churches) |
| GET | `/api/churches/:id/checkins` | User | Church visitors |
| GET | `/api/churches/:id/loyal-users` | User | Most loyal users |
| POST | `/api/churches/:id/favorite` | User | Add to favorites |
| DELETE | `/api/churches/:id/favorite` | User | Remove from favorites |
| GET | `/api/favorites` | User | List favorites |

### Suggestions & Admin
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/suggestions` | User | Submit (new_church/edit/schedule/general) |
| GET | `/api/suggestions/mine` | User | My suggestions |
| GET | `/api/suggestions` | Mod | All suggestions (status filter) |
| PUT | `/api/suggestions/:id` | Mod | Approve/reject with note |
| GET | `/api/admin/users` | Admin | List users |
| PUT | `/api/admin/users/:id/role` | Admin | Set role |
| PUT | `/api/admin/churches/:id/verify` | Admin | Verify church |
| DELETE | `/api/admin/churches/:id` | Admin | Delete church |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `wic` | Database user |
| `DB_PASSWORD` | `wic_dev_password` | Database password |
| `DB_NAME` | `whereischurch` | Database name |
| `DB_SSLMODE` | `disable` | SSL mode (`require` in production) |
| `JWT_SECRET` | - | **Required**: 32+ char random string |
| `JWT_EXPIRY_HOURS` | `24` | Token lifetime |
| `PORT` | `8080` | Server port |
| `ENVIRONMENT` | `development` | `production` enables strict validation |
| `COOKIE_SECURE` | `false` | `true` in production (HTTPS only) |

## Security

- JWT with HMAC-SHA256 algorithm validation (prevents `alg:none` attacks)
- Bcrypt password hashing with 72-byte truncation guard
- SameSite=Lax cookies prevent CSRF
- Typed input structs prevent mass assignment
- Context propagation enables request-scoped DB timeouts
- Configurable Secure cookie flag for HTTPS enforcement
- Production config validation blocks weak secrets and insecure defaults

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup instructions, code standards, and submission guidelines.

## License

This project is licensed under the terms in [LICENSE](LICENSE).
