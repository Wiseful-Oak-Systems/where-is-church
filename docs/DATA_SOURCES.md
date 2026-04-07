# Data Sources

## Current Sources

### OpenStreetMap (Primary)
- **Coverage**: ~70,000 places of worship in Brazil
- **Access**: Free via Overpass API
- **Data**: Name, coordinates, address, phone, website, denomination tag
- **Limitation**: Community-contributed — many parishes missing, especially rural. Denomination tags often empty (our smart classifier handles this).
- **Usage**: `make seed-gen` downloads and saves as `seeds/br.json.gz`

### MassTimes.org API (Catholic-specific)
- **Coverage**: 117,000+ churches in 201 countries, including Brazil
- **Access**: Free API key — request at https://apiv4.updateparishdata.org/
- **Data**: Name, address, coordinates, mass times, diocese, phone, email, website
- **Limitation**: Crowd-sourced, some entries may be outdated
- **Usage**: Set `MASSTIMES_API_KEY` in `.env`, then the seeder can enrich data
- **Status**: Integration built, awaiting API key

### User Contributions (Community)
- **Coverage**: Fills gaps that OSM and MassTimes miss
- **Access**: Built into the app via suggestion/proposal system
- **Data**: Whatever the user knows (name, address, coordinates via map click or auto-geocode)
- **Quality**: Reviewed by moderators, auto-approved for trusted users

## How Competitors Built Their Data

### Catholic Mass Times (130K+ churches, 136 countries)
Built over **11 years** through:
1. Manual founder entry (200 churches in Buenos Aires to bootstrap)
2. Crowdsourcing from 75,000+ users (5-10K updates/month)
3. **Regional app developers merging their databases** — the key scaling mechanism
4. Algorithmic detection of stale entries + manual verification

There is **no single bulk data source** for all Catholic parishes worldwide.

### MassTimes.org (117K+ churches, 201 countries)
- Run by Diocese of Lansing (Michigan, USA)
- Data authenticated by dioceses and religious orders
- Powers USCCB's official mass times page
- Volunteer-maintained with diocesan cooperation

## Sources That Do NOT Have Individual Parish Data

| Source | What it has | Why it doesn't help |
|--------|-----------|---------------------|
| Vatican (Annuario Pontificio) | 3,000+ dioceses | Diocese-level only, no parishes, paid |
| Catholic GeoHub (GoodLands/ArcGIS) | Diocese boundaries | Point data deliberately withheld for security |
| GCatholic.org | 225K parish count by diocese | Aggregate counts only, no addresses |
| CARA (Georgetown) | US diocesan statistics | Research data, not a parish directory |
| CNBB (Brazil bishops) | Diocese-level org data | No centralized parish list |

## For US-Only Data

| Source | Coverage | Cost |
|--------|----------|------|
| Official Catholic Directory (OCD) | 60K+ US entities | $225-395/yr digital, bulk license negotiated |
| catholicdata.co | 17,600 US parishes | ~$50-200 one-time Excel |
| catholicparishdirectory.com | 14K US parishes | Moderate one-time |

## Our Strategy

1. **OSM for initial coverage** (coordinates + names) — already integrated
2. **MassTimes.org API for Catholic enrichment** (mass times, verified data) — built, awaiting key
3. **Community contributions for completeness** (fills what both miss) — fully operational
4. **Smart denomination classifier** (prevents "Igreja Evangélica" from being tagged Catholic) — implemented
5. **Data quality loop** (check-ins → confirmations → community verified) — implemented

The goal is not to have perfect data at launch, but to have the **best community infrastructure** so data improves faster than competitors.
