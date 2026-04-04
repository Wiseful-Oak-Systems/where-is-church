# Architecture Decisions

## Database Migrations

### Decision: SQL-based migrations alongside GORM AutoMigrate

**Context**: GORM's `AutoMigrate` is convenient for development but has critical limitations:
- No rollback support
- No version tracking
- Silently drops columns in some drivers
- Cannot handle complex schema changes (renames, data migrations)

**Decision**: We provide both approaches:
1. **Development**: GORM `AutoMigrate` runs on startup for rapid iteration
2. **Production**: Use [golang-migrate](https://github.com/golang-migrate/migrate) with versioned SQL files in `migrations/`

**Migration files** follow the convention: `{sequence}_{description}.{up|down}.sql`

### Running Migrations

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
make migrate-up

# Rollback last migration
make migrate-down

# Create a new migration
make migrate-create NAME=add_notifications
```

## API Design

### Authentication
- JWT with HMAC-SHA256, algorithm validation, issuer claims
- Dual delivery: cookie (SameSite=Lax) for web UI, Bearer header for API clients
- 24-hour default expiry (configurable)

### Authorization Model
Three roles with escalating permissions:

```
User       → Search, check-in, suggest, favorite
Moderator  → + Edit churches, manage schedules, review suggestions
Admin      → + Manage users, verify/delete churches, set roles
```

Moderators are promoted by admins from loyal users (those with high check-in counts).

### Geospatial Queries
- Church search uses the **Haversine formula** in SQL for accurate distance calculation
- PostGIS extension is enabled but not yet used for spatial indexing (future optimization)
- Radius is capped at 100km to prevent full-table scans

### Data Quality Loop

```
User check-in → Verifies church exists and is active
                → Implicit schedule validation
                → Builds loyalty score
                → Admins promote loyal users to moderators
                → Moderators review user suggestions
                → Suggestions improve church data
                → Better data attracts more users
```

This self-reinforcing loop is our primary differentiator vs. competitors who rely on manual volunteer updates.

## Frontend Architecture

### Technology Choices
- **Vanilla JS** (no frameworks) — keeps bundle small, loads fast, no build step
- **Leaflet.js** with OpenStreetMap — free, open-source, no API key required
- **Go HTML templates** — server-rendered pages, no SPA complexity
- **CSS custom properties** — dark mode via `prefers-color-scheme` media query

### Mobile UX
- **Bottom sheet** pattern on mobile (like Google Maps) replaces sidebar
- **Touch gesture** handling for sheet drag
- **PWA manifest** for "Add to Home Screen"
- **Responsive breakpoint** at 768px

### Accessibility (WCAG 2.1)
- Skip-to-content link
- `aria-live` region for screen reader announcements of map events
- Keyboard-navigable markers
- Color-coded denomination badges with text labels (not color-only)
- Focus-visible outlines
- Semantic HTML5 with ARIA landmarks

## Schedule Types

The `MassSchedule` model supports three types:
- `mass` — Regular mass times
- `confession` — Confession/reconciliation schedules (with time ranges)
- `adoration` — Eucharistic adoration schedules (with time ranges)

This covers the top 3 most-searched sacrament types per our market research.
