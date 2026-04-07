# Market Research: Church Finder Applications (2025-2026)

## Competitive Landscape

### 1. Catholic Mass Times (catholicmasstimes.com)
**Market leader** — 130,000+ churches in 136 countries, 1.9M+ downloads.

| Feature | Available |
|---------|-----------|
| Mass/Confession/Adoration times | Yes |
| Live-stream links | Yes |
| Language filters (English/Latin/Spanish) | Yes |
| Map search | Yes |
| Community-driven corrections (75K+ contributors) | Yes |
| Favorites/bookmarking | Coming soon |
| Check-ins / loyalty | No |
| Ratings/reviews | No |

**Strengths**: Massive dataset, active contributor community (~10K updates/month), ad-free.
**Weaknesses**: No social features, no attendance tracking, no gamification.

### 2. MassTimes.org
117,000 churches in 201 countries.

| Feature | Available |
|---------|-----------|
| Mass/Confession/Adoration search | Yes |
| Map view | Yes |
| Favorites | No |
| Push notifications | No |
| Modern UI | No |

**Strengths**: Global coverage.
**Weaknesses**: App not updated in years, frequently broken on newer OS versions, no favorites, dated UI, no push notifications. Data accuracy depends on volunteer parish updates with no incentive structure.

### 3. ChurchFinder.com
280,000+ Christian churches (all denominations).

| Feature | Available |
|---------|-----------|
| Denomination filter | Yes |
| Photo tours / video profiles | Yes (premium) |
| Enhanced church profiles | Yes (paid parishes) |
| Sacrament-specific search | No |
| Attendance features | No |

**Strengths**: Largest multi-denomination dataset, 2M+ users/year.
**Weaknesses**: More directory than live-schedule tool. No sacrament search, no attendance features. Protestant-leaning.

### 4. Hallow
\#1 Catholic prayer app globally.

| Feature | Available |
|---------|-----------|
| Guided prayer / Rosary / Lectio Divina | Yes |
| Community pages | Yes |
| Parish partnership program | Yes |
| Prayer challenges/groups | Yes |
| Mass finding | No |

**Strengths**: Very strong engagement and social model, excellent design.
**Weaknesses**: Not a mass finder — complementary product, not a competitor.

### 5. Flocknote
Parish-facing communication and giving platform.

| Feature | Available |
|---------|-----------|
| Email/text e-bulletins | Yes |
| Online giving | Yes |
| RSVP/polls | Yes |
| Ministry groups | Yes |
| Consumer-facing search | No |

**Strengths**: Owns the "parish bulletin + donation" use case.
**Weaknesses**: B2B (parishes pay), not consumer-facing.

---

## Feature Gap Analysis

### Priority Features (by user demand)

| Priority | Feature | Competitors | Our Status |
|----------|---------|------------|------------|
| 1 | **Favorites / Bookmarks** | None have it (CMT "coming soon") | **Implemented** |
| 2 | **Confession & Adoration Times** | CMT, MassTimes have it | **Implemented** |
| 3 | **Data Freshness Indicators** | None show "last verified" | **Implemented** |
| 4 | **Push Notifications / Reminders** | None have nailed it | Planned |
| 5 | **Live-Stream Links** | CMT has it | Planned |
| 6 | **Language / Rite Filters** | CMT has basic | Planned |
| 7 | **Holy Day Special Schedules** | None handle gracefully | Planned |
| 8 | **Navigation Integration** | None have 1-tap directions | Planned |
| 9 | **Parish Photos** | ChurchFinder (premium) | Planned |
| 10 | **Dark Mode** | None | Planned |

### UX Improvements (based on competitor complaints)

1. **Stale data** — The #1 complaint. Our check-in + suggestion + moderator workflow already addresses this. Surfacing "last verified" dates and "confirm times are correct" prompts will make it explicit.

2. **App abandonment after OS updates** — MassTimes.org app was broken for years. Our web-first responsive approach with PWA support avoids app store dependency.

3. **Search radius rigidity** — Competitors default to nearest 30 results with no control. Our prominent radius slider (1-50km) is already a differentiator.

4. **No confirmation of data accuracy** — Our check-in system implicitly validates data ("I was here and the times were right"). Making this explicit with post-visit prompts is a unique advantage.

5. **Cluttered map with no filtering** — Competitors show all pins at once. Our denomination filter + schedule type filter (mass/confession/adoration) provides better signal-to-noise.

---

## Our Unique Differentiators

### 1. Check-in Loyalty System
No competitor has Foursquare-style attendance tracking. This drives repeat engagement and creates social proof (most visited churches surface naturally). Loyal users earn moderator status — a self-sustaining quality loop.

### 2. Community-Promoted Moderation
Loyal users earn moderator status based on check-in activity, creating a self-sustaining data quality loop. Competitors rely on anonymous volunteer pipelines with no incentive structure.

### 3. Structured Suggestion Workflow
Submit → mod review → publish pipeline vs. competitors' "email us and hope" approach. Faster corrections, higher trust.

### 4. Multi-Denomination from Day One
Catholic Mass Times is Catholic-only. ChurchFinder is Protestant-leaning. We support Catholic, Orthodox, Protestant, Anglican, and Evangelical in one app with denomination-aware defaults.

### 5. Open-Source + Modern Stack
Go + PostGIS + Leaflet.js enables diocese-level data integrations, geospatial queries at scale, and community contributions that apps built on outdated Cordova/PhoneGap stacks cannot easily support.

---

## Market Opportunity

- Catholic Mass Times reached 2M downloads with a basic feature set — proving strong demand.
- No app combines **mass finding + social check-ins + community moderation**.
- The stale data problem is universally unsolved — our check-in verification loop is the first real answer.
- Multi-denomination support opens a much larger TAM than Catholic-only apps.
