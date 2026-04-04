# Contributing to Where is Church?

Thank you for your interest in contributing! This project helps people find churches and mass times near them. Every contribution — from bug reports to code changes — makes a real difference.

## Getting Started

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Make

### Local Setup

```bash
git clone https://github.com/Wiseful-Oak-Systems/where-is-church.git
cd where-is-church
cp .env.example .env
make db-up   # starts PostgreSQL + PostGIS
make run     # starts the server at http://localhost:8080
```

### Running Tests

```bash
make test          # run all tests
make test-verbose  # run with verbose output
make lint          # run linters (requires golangci-lint)
```

## How to Contribute

### Reporting Bugs

- Open an issue with a clear title and description
- Include steps to reproduce
- Include expected vs actual behavior
- Include your Go version (`go version`) and OS

### Suggesting Features

- Check existing issues first to avoid duplicates
- Open an issue tagged `enhancement`
- Describe the use case, not just the solution

### Submitting Code

1. **Fork** the repository
2. **Create a branch** from `main` (`git checkout -b feature/my-feature`)
3. **Write tests** — all new features need tests with BDD-style descriptions
4. **Run the full test suite** — `make test`
5. **Commit** with a clear message following [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` for new features
   - `fix:` for bug fixes
   - `refactor:` for code improvements
   - `test:` for test additions
   - `docs:` for documentation
6. **Push** and open a Pull Request

## Code Standards

### Go Code

- Follow [Effective Go](https://go.dev/doc/effective_go) and the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- Use `gofmt` (enforced by CI)
- All exported functions need godoc comments
- Error handling: always check errors, never silently discard
- Context: propagate `context.Context` through all DB calls
- Validation: validate all user input at the handler level
- Security: see [SECURITY.md](SECURITY.md) for security practices

### Test Style

We use **BDD-style test descriptions** that a Product Manager can read. Example:

```go
func TestCheckIn(t *testing.T) {
    t.Run("User can check in at a church like Foursquare, marking they attended", func(t *testing.T) {
        // ...
    })

    t.Run("User cannot check in at the same church twice within 2 hours", func(t *testing.T) {
        // ...
    })
}
```

### Frontend

- Vanilla JS (no frameworks — keeps it lightweight and fast)
- Mobile-first responsive CSS with CSS custom properties
- Leaflet.js for maps (OpenStreetMap tiles)
- Semantic HTML5 with ARIA attributes for accessibility

## Project Structure

```
cmd/server/          # Application entrypoint
internal/
  config/            # Configuration loading and validation
  database/          # Database connection and migrations
  handlers/          # HTTP handlers (API endpoints)
  middleware/        # JWT auth, role-based access control
  models/            # GORM models and domain types
  testutil/          # Shared test helpers
web/
  static/css/        # Stylesheets
  static/js/         # Client-side JavaScript
  templates/         # Go HTML templates
docs/                # Documentation
```

## Need Help?

- Open an issue with the `question` label
- Check existing issues and discussions
- Read the [docs/](docs/) directory for architecture decisions
