# Product Roadmap

## Implemented (Current Release)

### Core Product
- [x] Interactive OpenStreetMap with Haversine search (1-100km radius)
- [x] Multi-denomination support (Catholic, Orthodox, Protestant, Anglican, Evangelical)
- [x] Mass, Confession, and Adoration schedule management
- [x] JWT authentication with role hierarchy (User → Moderator → Church Owner → Community Manager → Admin)
- [x] Foursquare-style check-ins with 2-hour dedup
- [x] Favorites/bookmarks
- [x] File attachments (local filesystem or S3)

### Community Governance
- [x] Trust/reputation scoring (0-1000, 5 levels)
- [x] Auto-approval for trusted contributors (score ≥500 for edits, ≥750 for schedules)
- [x] Rate limiting for new users (3 suggestions/day for trust <100)
- [x] Rejection streak suspension (3 consecutive = 48h cooldown)
- [x] Church ownership claims with evidence review
- [x] Data quality states: Unverified → Community Confirmed → Officially Verified
- [x] Independent user confirmations (3 needed for community_confirmed)
- [x] Contribution impact counter ("Your edits helped X people find Mass")
- [x] Community leaderboard

### Admin Tools
- [x] Audit trail with actor tracking
- [x] Bulk verify churches (1-100 at once)
- [x] Bulk review suggestions
- [x] Duplicate church detection (within 100m)
- [x] Church merge (transfers all related data)
- [x] Data quality health dashboard
- [x] Stale data identification (>90 days)

### Developer Experience
- [x] GitHub Actions CI (test, lint, build, Docker)
- [x] Multi-stage Dockerfile (Alpine, non-root, ~25MB)
- [x] golangci-lint v2 with security linters
- [x] Pre-commit hooks
- [x] SQL migrations (golang-migrate compatible)
- [x] Swagger/OpenAPI annotations
- [x] 211 BDD tests

### UI/UX
- [x] Dark mode (prefers-color-scheme)
- [x] Mobile bottom sheet (drag gesture)
- [x] Address search via Nominatim
- [x] WCAG 2.1 accessibility (skip link, aria-live, keyboard markers)
- [x] Denomination-colored map markers
- [x] PWA manifest

---

## Phase 2: Engagement Engine (Next)

*The liturgical calendar is our built-in re-engagement engine — exploit it before anything else.*

### Liturgical Calendar Integration
- [ ] Liturgical calendar API (Advent, Lent, Holy Days, Saint feast days)
- [ ] Push notification reminders tied to the calendar
- [ ] Special schedule display for Holy Days (Christmas vigil, Easter Triduum, Ash Wednesday)
- [ ] Seasonal banner/theme in the UI during Lent, Advent, etc.

### Streak System (Hallow Model)
- [ ] Daily prayer/Mass attendance streak counter
- [ ] Streak display on profile with flame emoji
- [ ] Streak milestone notifications (7 days, 30 days, 100 days, 365 days)
- [ ] "Don't break your streak" push notification

### Seasonal Community Challenges (Hallow Pray40 Model)
- [ ] "Attend Mass every day of Lent" challenge
- [ ] Community-wide participation counter
- [ ] Challenge leaderboard
- [ ] Completion badge/certificate

### Limited-Time Collectible Badges
- [ ] Christmas Mass badge
- [ ] Easter Sunday badge
- [ ] Ash Wednesday badge
- [ ] Patronal feast day badges (location-specific)
- [ ] First check-in badge, 10th church badge, etc.

---

## Phase 3: Social Layer

### Friend System
- [ ] Add friends / contacts
- [ ] Weekly friend leaderboard (Foursquare model — social competition is stickier than global)
- [ ] Share check-in with friends
- [ ] See which friends attend the same church

### Parish Community Pages
- [ ] Prayer intention sharing within a parish (Hallow model)
- [ ] Parish bulletin integration (MassTimes model)
- [ ] Church owner pinned announcements (e.g., "Mass times changed for Advent")
- [ ] Parish event calendar

### Notifications
- [ ] Mass time reminders (30 min before)
- [ ] Nearby church discovery notification (traveling users)
- [ ] When a church you follow updates schedules
- [ ] Weekly digest of community activity

---

## Phase 4: Data Intelligence

### AI-Assisted Data
- [ ] Auto-parse church websites for Mass times (web scraping + LLM)
- [ ] Discrepancy detection (scraped vs. user-reported times)
- [ ] Smart schedule suggestions based on check-in patterns
- [ ] AI moderation assistance for suggestion review

### Crowdsourced Schedule Validation
- [ ] If 5+ users check in at an unlisted time → auto-flag for review
- [ ] Passive data collection: attendance heatmaps from check-in timestamps
- [ ] "Were the times accurate?" post-visit prompt

### Advanced Search
- [ ] "Mass happening now" filter (matches current time against schedules)
- [ ] "Confession available today" filter
- [ ] Language-specific Mass search (Latin, Spanish, Vietnamese, etc.)
- [ ] Wheelchair accessible filter
- [ ] Parking available filter

---

## Phase 5: Platform Expansion

### Moderator Tools
- [ ] Moderator training pipeline with test suggestions
- [ ] Review queue with SLA tracking (72h warning)
- [ ] Conflict/edit war detection (3 reverts on same field = auto-lock)
- [ ] Vandalism/spam auto-detection rules
- [ ] CSV export of pending queue

### Church Owner Features
- [ ] Structured evidence types for claims (website URL, letterhead photo, directory listing)
- [ ] "Verified by Parish" badge on listing
- [ ] Owner dashboard with visitor analytics
- [ ] Direct schedule management (no approval needed)
- [ ] Donation/giving links

### Integration
- [ ] Ride-sharing button (Uber/Lyft to church — Catholic Mass Times model)
- [ ] Navigation integration (Google Maps / Apple Maps / Waze deep links)
- [ ] Parish management system integrations (Flocknote, myParish)
- [ ] Diocese data import/export API
- [ ] iCal/Google Calendar subscription for schedules

---

## Competitive Intelligence Sources

See [docs/MARKET_RESEARCH.md](MARKET_RESEARCH.md) for our full competitive analysis.

Key competitors monitored:
- **Catholic Mass Times** — 130K+ churches, community corrections, Uber integration
- **MassTimes.org** — 121K churches, bulletin links, not innovating
- **Hallow** — Liturgical calendar + streaks as retention engine
- **Foursquare/Swarm** — Social competition > individual achievement
- **Google Local Guides** — Visible badges as permanent status display
- **iNaturalist** — Research Grade 3-state quality system
- **Waze** — Passive contribution + impact messages for retention
- **Nextdoor** — 300K volunteer moderators as engagement mechanism
