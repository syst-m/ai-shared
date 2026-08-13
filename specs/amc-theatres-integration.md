# AMC Theatres API Integration Specification

**Status:** Draft  
**Author:** AI Agent (on behalf of Frank Ellis)  
**Date:** 2026-08-11  
**Target Repo:** `syst-m/ai-shared`  

---

## Table of Contents

1. [Overview](#1-overview)
2. [Integration Architecture](#2-integration-architecture)
3. [API Authentication](#3-api-authentication)
4. [Core API Endpoints](#4-core-api-endpoints)
   - [Movie Search & Details](#41-movie-search--details)
   - [Theater Locator](#42-theater-locator)
   - [Showtime Retrieval](#43-showtime-retrieval)
   - [Seat Selection & Reservation Booking](#44-seat-selection--reservation-booking)
5. [Data Models & Schemas](#5-data-models--schemas)
6. [Error Handling & Retry Strategy](#6-error-handling--retry-strategy)
7. [Rate Limiting Compliance](#7-rate-limiting-compliance)
8. [Seat Selection Strategy](#8-seat-selection-strategy)
9. [Testing Strategy](#9-testing-strategy)
10. [Implementation Plan](#10-implementation-plan)

---

## 1. Overview

This specification defines the integration between an AI assistant layer and the AMC Theatres public REST API suite. The goal is to enable natural-language movie discovery, showtime lookup, seat selection, and ticket reservation booking for the end user.

**Scope:**
- Movie search and metadata retrieval
- Theater discovery by location or identifier
- Showtime queries (by date, theater, movie, or proximity)
- Seating layout inspection and algorithmic seat selection
- Order creation, confirmation, and fulfillment

The AMC API base URL for production is `https://api.amctheatres.com`. All endpoints are versioned via URI path prefix (e.g., `/v2/`, `/v3/`). Responses use a HAL-style JSON envelope with `_embedded` collections and `_links` navigation.

---

## 2. Integration Architecture

```
┌──────────────┐     ┌─────────────────────┐     ┌──────────────────┐
│   User       │────▶│   AI Assistant      │────▶│  AMC Theatres    │
│  (Telegram/  │◀────│   Layer             │◀────│  REST API        │
│   Chat)      │     │                     │     │                  │
└──────────────┘     └─────────────────────┘     └──────────────────┘
                             │
                            ┌┴────────────────────────┐
                            │  Internal Service Layer │
                            ├─────────────────────────┤
                            │ • Auth Manager          │
                            │ • Rate Limiter          │
                            │ • Retry Handler         │
                            │ • Cache Layer (optional)│
                            │ • Seat Selection Engine │
                            └─────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility |
|---|---|
| **AI Assistant Layer** | Parses user intent, routes to correct API flow, formats responses for the user |
| **Auth Manager** | Stores and injects `X-AMC-Vendor-Key`; refreshes tokens for e-commerce APIs |
| **Rate Limiter** | Enforces per-endpoint request budgets; implements token-bucket throttling |
| **Retry Handler** | Implements exponential backoff on 429/5xx responses with jitter |
| **Cache Layer** (optional) | Caches movie catalog and theater lists (TTL 1–6 hours); never cache showtimes/seats |
| **Seat Selection Engine** | Scores available seats using configurable criteria; returns ranked recommendations |

### Design Principles

- The integration layer MUST be stateless where possible, persisting only user preferences.
- All external API calls SHOULD pass through the rate limiter before dispatch.
- Movie and theater catalogs MAY be cached; showtimes and seating availability MUST NOT be cached beyond 60 seconds.
- PII (AMC account data, payment info) MUST never be logged or stored in plaintext.

### Security

- All user-supplied parameters (theater IDs, movie IDs, seat IDs, dates) MUST be validated against expected types and ranges before being forwarded to the AMC API.
- Theater IDs MUST be positive integers; movie IDs MUST be positive integers; seat IDs MUST match the pattern `^[A-Z][0-9]+$` (single letter + number, e.g., "J12").
- The integration MUST sanitize natural-language input to prevent parameter injection (e.g., a user saying "book seat J12; DROP TABLE seats" MUST NOT result in the injection reaching any downstream system).
- When operating as an AI agent, the integration MUST require explicit user confirmation before executing any write operation (order creation, seat selection, fulfillment).
- API keys and auth tokens MUST be stored in a secrets manager or encrypted at rest; they MUST NEVER appear in logs, error messages, environment variable dumps, or version control.
- The integration SHOULD implement request signing or CSRF-like tokens if exposed via a web interface, to prevent request forgery.

---

## 3. API Authentication

### 3.1 Vendor API Key (Read-Only APIs)

All requests to Showtimes (v2), Movie (v2), Theatre (v2), and Location (v2) APIs require a vendor API key passed as an HTTP header:

```
X-AMC-Vendor-Key: {api-key}
```

- The key is obtained by registering on the [AMC Developer Portal](https://developers.amctheatres.com/GettingStarted/NewVendorRequest).
- The integration layer MUST store the key in a secure secret manager or encrypted configuration file.
- The key MUST NOT appear in logs, error messages, or version control.

### 3.2 Auth Token (E-Commerce APIs)

Seating (v3), Order (v3), and Barcode (v3) APIs require additional authentication via an `X-AMC-Auth-Token` header alongside the vendor key:

```
X-AMC-Vendor-Key: {api-key}
X-AMC-Auth-Token: {auth-token}
```

- Access to these endpoints requires explicit partner approval and a contractual agreement with AMC.
- The integration layer SHOULD support both authenticated and unauthenticated flows, gracefully degrading read-only functionality when e-commerce credentials are unavailable.

### 3.3 Authentication Flow

1. **Provisioning:** Register on the AMC Developer Portal → receive `X-AMC-Vendor-Key`.
2. **E-Commerce Approval** (optional): Apply for seating/order API access → receive `X-AMC-Auth-Token`.
3. **Runtime Injection:** Auth Manager injects headers into every outbound HTTP request.

```
┌─────────────┐   ┌──────────────────┐   ┌─────────────────┐
│  Register    │──▶│ Receive Vendor   │──▶│ Inject Header   │
│  (Developer  │   │ Key + Auth Token │   │ on Every Request│
│  Portal)     │   │                  │   │                 │
└─────────────┘   └──────────────────┘   └─────────────────┘
```

---

### 4.0 HAL Envelope Structure

All AMC API responses use a HAL (Hypertext Application Language) envelope. A typical response wraps the resource in `_embedded` and provides navigation via `_links`:

```json
{
  "_embedded": { "movies": [ /* array of movie objects */ ] },
  "_links": { "self": { "href": "/v2/movies?page-number=1" } }
}
```

## 4. Core API Endpoints

### 4.1 Movie Search & Details

**API:** Movie API v2  
**Base Path:** `/v2/movies`

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v2/movies` | List currently playing and upcoming movies |
| `GET` | `/v2/movies/{movie-id}` | Get details for a specific movie |

**Query Parameters (List):**

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `page-number` | integer | No | 1 | Page of results |
| `page-size` | integer | No | 10 | Results per page (max 100) |

**Response Schema — Movie:**

```json
{
  "id": 98765,
  "name": "Movie Title",
  "slug": "movie-title",
  "sortName": "Movie Title",
  "runtime": 132,
  "mpaaRating": "PG-13",
  "genre": "Action, Adventure",
  "tagline": "Coming this summer.",
  "synopsis": "A brief description...",
  "attributes": [
    { "id": 1, "code": "IMAX", "name": "IMAX" },
    { "id": 2, "code": "DOLBY_CINEMA", "name": "Dolby Cinema" }
  ],
  "media": {
    "posterImage": "https://...",
    "heroImage": "https://..."
  },
  "_links": {
    "self": { "href": "/v2/movies/{movie-id}" }
  }
}
```

### 4.2 Theater Locator

**API:** Theatre API v2 / Location API v2  
**Base Path:** `/v2/theatres` and `/v2/locations`

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v2/theatres/{theatre-number}` | Get details for a specific theater |
| `GET` | `/v2/theatres` | List all active theaters |
| `GET` | `/v2/locations/state/{state-name}` | Theaters by state |
| `GET` | `/v2/locations/name/{theatre-name}` | Theaters by name (fuzzy match) |
| `GET` | `/v2/locations/nearby?latitude={lat}&longitude={lon}` | Theaters near coordinates |

**Response Schema — Theatre:**

```json
{
  "id": 1234,
  "number": "1234",
  "name": "AMC City National 10",
  "slug": "amc-city-national-10-imax-1234",
  "marketId": 5678,
  "marketName": "Los Angeles",
  "address": {
    "streetAddress": "550 S Hope St",
    "city": "Los Angeles",
    "state": "CA",
    "zip": "90071"
  },
  "geo": {
    "latitude": 34.0485,
    "longitude": -118.2574
  },
  "phone": "(213) 555-0100",
  "hasIMAX": true,
  "hasDolbyCinema": true,
  "hasPrime": true,
  "_links": { ... }
}
```

### 4.3 Showtime Retrieval

**API:** Showtimes API v2  
**Base Path:** `/v2/showtimes` and `/v2/theatres/{theatre-number}/showtimes`

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v2/theatres/{theatre-number}/showtimes` | All future showtimes for a theater |
| `GET` | `/v2/theatres/{theatre-number}/showtimes/{date}` | Showtimes on a specific date |
| `GET` | `/v2/theatres/{theatre-number}/showtimes/{date}/views/embargoed` | Embargoed showtimes |
| `GET` | `/v2/theatres/{theatre-number}/movies/{movie-id}/earliest-showtime` | Earliest showing of a movie |
| `GET` | `/v2/showtimes/{id}` | Showtime by ID |
| `GET` | `/v2/showtimes/views/current-location/{date}/{latitude}/{longitude}` | Showtimes near location |

**Query Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `movie-id` | integer | Filter to a specific movie |
| `include-attributes` | string | Comma-delimited attributes to include (e.g., `IMAX,DOLBY_CINEMA`) |
| `exclude-attributes` | string | Comma-delimited attributes to exclude |
| `attribute-operator` | string | `and` (all must match) or `or` (any matches) |
| `page-number` | integer | Page number (default 1) |
| `page-size` | integer | Results per page (max 100, default 10) |

**Response Schema — Showtime:**

```json
{
  "id": 12345678,
  "movieId": 98765,
  "movieName": "Movie Title",
  "showDateTimeUtc": "2026-08-15T21:00:00Z",
  "showDateTimeLocal": "2026-08-15T14:00:00-07:00",
  "utcOffset": "-07:00",
  "theatreId": 1234,
  "auditorium": 3,
  "layoutId": 99001,
  "performanceNumber": 556789,
  "runTime": 132,
  "mpaaRating": "PG-13",
  "genre": "Action",
  "purchaseUrl": "https://www.amctheatres.com/movies/...",
  "mobilePurchaseUrl": "https://m.amctheatres.com/...",
  "isAlmostSoldOut": false,
  "isSoldOut": false,
  "isCanceled": false,
  "isPrivateRental": false,
  "isDiscountMatineePriced": true,
  "ticketPrices": [
    {
      "priceType": "Admit",
      "sku": "ADV-ADMIT",
      "price": 16.50,
      "tax": 1.24,
      "formattedPrice": "$16.50",
      "formattedTax": "$1.24"
    }
  ],
  "attributes": [
    { "id": 1, "code": "IMAX", "name": "IMAX" },
    { "id": 2, "code": "LASED", "name": "Digital" }
  ],
  "_links": { ... }
}
```

### 4.4 Seat Selection & Reservation Booking

#### Seating API v3

**Base Path:** `/v3/seating-layouts`

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v3/seating-layouts/{theatreNumber}/{performanceNumber}` | Get seating layout for a performance |
| `GET` | `/v3/seating-layouts/{theatreNumber}/auditoriums` | List all auditorium layouts at a theater |

**Response Schema — Seating Layout:**

```json
{
  "theatreNumber": 1234,
  "performanceNumber": 556789,
  "layoutId": 99001,
  "auditoriumName": "3",
  "rows": [
    {
      "rowId": "A",
      "seats": [
        { "seatId": "A1", "label": "A1", "status": "Available" },
        { "seatId": "A2", "label": "A2", "status": "Reserved" },
        { "seatId": "A3", "label": "A3", "status": "Unavailable" }
      ]
    }
  ],
  "seatStatuses": {
    "Available": "Available",
    "Reserved": "Reserved",
    "Unavailable": "Unavailable",
    "Selected": "Selected"
  }
}
```

**Seat Status Values:**
> **Note:** The API returns seat `status` as a **string** (e.g., `"Available"`). The numeric codes below are for internal mapping only and must not be sent to the API.
| Value | Label | Description |
|---|---|---|
| `0` | Available | Seat can be booked |
| `1` | Reserved | Currently held by another transaction |
| `2` | Unavailable | Not a valid seat (aisle, wall, blocked) |
| `3` | Selected | Marked for booking in current session |

### 4.4.1 Order Update

**Base Path:** `/v3/orders/{orderToken}`

| Method | Endpoint | Description |
|--------|----------|-------------|
| `PUT` | `/v3/orders/{orderToken}` | Update an existing order |

**Update Request:**

```json
{
  "items": [
    {
      "itemType": "TICKET",
      "sku": "ADV-ADMIT",
      "performanceNumber": 556789,
      "quantity": 3,
      "seats": ["J12", "J13", "J14"]
    }
  ]
}
```

Only the `items` array can be updated. Seat changes trigger a re-validation of availability. Quantity changes are subject to the same ~10-minute expiration window.

### 4.4.2 Embargoed Showtimes

**Endpoint:** `/v2/theatres/{theatre-number}/showtimes/{date}/views/embargoed`

Returns showtimes for movies not yet publicly announced (e.g., advance industry screenings). Requires the same `X-AMC-Vendor-Key` header as other showtime endpoints. The response schema is identical to the standard showtime endpoint (Section 4.3). This endpoint is primarily useful for internal monitoring and pre-release planning; the AI assistant layer SHOULD ignore embargoed showtimes unless explicitly requested by the user with a valid reason.

### 4.4.3 Order Creation to Fulfillment Flow

The complete order lifecycle consists of the following steps:

1. **Seat selection** — User selects or accepts recommended seats via the Seat Selection Engine (§8).
2. **Order creation** — POST to `/v3/orders` with selected seats. The API returns an order with status `Created` and a token.
3. **Seat re-check** — Before fulfillment, re-query the seating layout to confirm all seats are still `Available`. If any seat is no longer available, re-run the seat selection engine or abort (§4.4, "Seat conflict handling").
4. **Order fulfillment** — POST to `/v3/orders/{orderToken}/fulfill` to submit for payment and barcode generation.
5. **Confirmation** — Poll GET `/v3/orders/{orderToken}` until status is `Confirmed`, or wait for the fulfillment response.

Orders have a ~10 minute window between creation and fulfillment. The integration layer MUST monitor order expiration and alert the user before timeout. If an order expires, the system MUST create a new order rather than retrying on the expired token.

**Seat conflict handling:** Before submitting an order for fulfillment, the integration MUST re-check seat availability via the Seating API. If any selected seat is no longer `Available`, the system MUST either (a) re-run the seat selection engine for remaining available seats, or (b) abort and inform the user. This prevents silent failures where seats were booked by another party between selection and fulfillment.

### 4.4.4 Order Status Values

| Status | Description |
|--------|-------------|
| `Created` | Order created but not yet submitted for payment |
| `Submitted` | Order submitted for payment processing |
| `Confirmed` | Payment successful; tickets/barcode generated |
| `Expired` | Order expired before fulfillment (10-minute window elapsed) |
| `Canceled` | Order canceled by user or system |

#### Order API v3

**Base Path:** `/v3/orders`

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v3/orders` | Create a new order (reserve tickets + seats) |
| `GET` | `/v3/orders/{orderToken}` | Get order details / check status |
| `PUT` | `/v3/orders/{orderToken}` | Update an existing order |
| `POST` | `/v3/orders/{orderToken}/fulfill` | Submit order for payment and fulfillment |

**Order Creation Request:**

```json
{
  "theatreNumber": 1234,
  "amcAccountId": "user-amc-account-id",
  "items": [
    {
      "itemType": "TICKET",
      "sku": "ADV-ADMIT",
      "performanceNumber": 556789,
      "quantity": 2,
      "seats": ["J12", "J13"]
    }
  ]
}
```

**Order Response:**

```json
{
  "orderId": 100001,
  "token": "abc123-token",
  "amcAccountId": "user-amc-account-id",
  "status": "Created",
  "createdUtc": "2026-08-15T19:00:00Z",
  "expirationUtc": "2026-08-15T19:10:00Z",
  "totalAmount": 33.00,
  "totalTax": 2.48,
  "totalConvenienceFees": 4.50,
  "items": [ ... ],
  "_links": {
    "self": { "href": "/v3/orders/abc123-token" },
    "fulfill": { "href": "/v3/orders/abc123-token/fulfill" }
  }
}
```

**Order Statuses:** `Created` → `Submitted` → `Confirmed` | `Expired` | `Canceled`

- Orders have a ~10 minute window between creation and fulfillment.
- The integration layer MUST monitor order expiration and alert the user before timeout.
- If an order expires, the system MUST create a new order rather than retrying on the expired token.
- **Seat conflict handling:** Before submitting an order for fulfillment, the integration MUST re-check seat availability via the Seating API. If any selected seat is no longer `Available`, the system MUST either (a) re-run the seat selection engine for remaining available seats, or (b) abort and inform the user. This prevents silent failures where seats were booked by another party between selection and fulfillment.

---

## 5. Data Models & Schemas

### 5.1 Internal Abstraction Layer

The integration SHOULD define internal data types that normalize AMC API responses:

```typescript
interface Movie {
  id: number;
  title: string;
  slug: string;
  runtimeMinutes: number;
  rating: string;        // e.g., "PG-13"
  genres: string[];
  formats: string[];     // e.g., ["IMAX", "Dolby Cinema"]
  posterUrl?: string;
}

interface Theatre {
  id: number;
  name: string;
  slug: string;
  address: TheatreAddress;
  city: string;
  state: string;
  zip: string;
  lat: number;
  lon: number;
  phone?: string;
  formats: string[];     // available format codes
}

interface TheatreAddress {
  streetAddress: string;
  city: string;
  state: string;
  zip: string;
}

interface Showtime {
  id: number;
  movieId: number;
  movieTitle: string;
  theatreId: number;
  auditorium: number;
  performanceNumber: number;
  layoutId: number;
  startTimeUtc: string;
  startTimeLocal: string;
  runtimeMinutes: number;
  rating: string;
  isSoldOut: boolean;
  isAlmostSoldOut: boolean;
  isCanceled: boolean;
  formats: string[];
  ticketPrices: TicketPrice[];
}

interface Seat {
  id: string;           // e.g., "J12"
  row: string;          // e.g., "J"
  number: string;       // e.g., "12"
  status: 'Available' | 'Reserved' | 'Unavailable' | 'Selected';
}

interface SeatLayout {
  theatreNumber: number;
  performanceNumber: number;
  auditoriumName: string;
  rows: SeatRow[];
}

interface SeatRow {
  rowId: string;
  seats: Seat[];
}

interface TicketPrice {
  type: string;         // e.g., "Admit", "Premium", "Child"
  sku: string;
  amount: number;
  tax: number;
  formattedPrice: string;
}
```

### 5.2 Error Response

All AMC API error responses follow the HAL-style `ApiError` schema:

```json
{
  "statusCode": 429,    // HTTP status code
  "message": "Rate limit exceeded",
  "errorCode": "RATE_LIMIT_EXCEEDED",
  "moreInfo": "Please reduce request frequency."
}
```

---

## 6. Error Handling & Retry Strategy

### 6.1 HTTP Status Code Handling

| Status | Meaning | Action |
|--------|---------|--------|
| `200` | Success | Process response |
| `400` | Bad Request | Log details; return user-friendly error; DO NOT retry |
| `401` | Unauthorized | Check API key validity; alert administrator |
| `403` | Forbidden | Endpoint requires elevated access; log and degrade gracefully |
| `404` | Not Found | Resource doesn't exist; return "not found" to user |
| `429` | Rate Limited | Apply exponential backoff + jitter retry (see §6.2) |
| `500–599` | Server Error | Apply exponential backoff + jitter retry up to 3 attempts |

### 6.2 Retry Policy
> **Note:** The `Retry-After` header, if present in a 429 response, MUST take precedence over the calculated backoff delay.

The integration MUST implement the following retry strategy:

1. **Initial delay:** 1 second
2. **Backoff multiplier:** 2× (exponential)
3. **Max attempts:** 3 retries (4 total requests)
4. **Jitter:** Add random delay of 0–500ms to each backoff interval
5. **Retryable statuses:** `429`, `502`, `503`, `504`
6. **Non-retryable statuses:** `400`, `401`, `403`, `404`

```
Attempt 1: immediate
Attempt 2: 1000ms ± [0, 500ms] jitter
Attempt 3: 2000ms ± [0, 500ms] jitter
Attempt 4: 4000ms ± [0, 500ms] jitter
→ After 4 failed attempts: return error to user
```

### 6.3 Timeout Configuration

| Operation | Timeout |
|---|---|
| Movie/Theater catalog queries | 5 seconds |
| Showtime queries | 8 seconds |
| Seating layout retrieval | 8 seconds |
| Order creation/fulfillment | 15 seconds |
| Connection timeout (all) | 3 seconds |

### 6.4 Circuit Breaker

- If 5 consecutive requests to the same endpoint fail with 5xx, open the circuit for 30 seconds.
- After 30 seconds, allow one probe request. If it succeeds, close the circuit; if it fails, reopen for another 30 seconds.

---

## 7. Rate Limiting Compliance

### 7.1 AMC Rate Limit Model

AMC uses a credit-based rate limiting system where **one credit equals one API call**. The exact per-second or per-minute quota is determined by the vendor agreement level. When the limit is exceeded, the API returns HTTP 429.

### 7.2 Token Bucket Implementation

The integration SHOULD implement a client-side token bucket:

| Parameter | Default Value | Configurable |
|---|---|---|
| Bucket capacity | 10 tokens | Yes |
| Refill rate | 1 token / 2 seconds | Yes |
| Per-request cost | 1 token | Yes |
| Behavior on empty | Queue and wait for refill | — |

> **Note:** The maximum queue wait time is 30 seconds. If the bucket does not have enough tokens within 30 seconds, the request is rejected with a `429` response to the user.

### 7.1.1 Rate Limit Credit Costs

AMC's credit-based rate limiting assigns different costs per endpoint:

| Endpoint Category | Credit Cost |
|-------------------|-------------|
| Movie, Theatre, Location (v2) | 1 credit |
| Showtimes (v2) | 1 credit |
| Seating (v3) | 2 credits |
| Order (v3) | 2 credits |

These are typical values; the actual costs are determined by the vendor agreement. The integration SHOULD log the `X-RateLimit-Remaining` header (if provided) to detect deviations from expected costs.

### 7.3 Adaptive Throttling

- The integration MUST monitor `429` responses and dynamically reduce the effective rate.
- On receiving a 429, halve the current request rate and hold for 10 seconds before resuming.
- After 60 seconds without a 429, gradually ramp up to the original rate.

### 7.4 Request Budgeting

| Use Case | Max Concurrent Requests | Notes |
|---|---|---|
| Movie catalog sync | 2 | Batch by page |
| Theater search | 1 | Sequential |
| Showtime lookup | 1 per theater | Serialize across theaters |
| Seat layout check | 1 | Single at a time |
| Order operations | 1 | Strictly sequential |

---

## 8. Seat Selection Strategy

The integration MUST implement a scoring algorithm for ranking available seats. The algorithm assigns a composite score (0–100) to each available seat based on configurable criteria.

### 8.1 Scoring Criteria

| Criterion | Weight | Description |
|---|---|---|
| **Center alignment** | 35% | Seats closer to the horizontal center of the auditorium score higher. Ideal = exact center column. |
| **Row depth** | 30% | Rows in the middle 60% of the auditorium (neither too close nor too far) score highest. The "golden zone" is typically rows 40–70% from the screen. |
| **Pair adjacency** | 15% | When booking multiple tickets, adjacent seats receive a bonus. Non-adjacent selections are penalized. |
| **Format premium** | 10% | IMAX/Dolby Cinema formats may have different optimal zones; adjust row depth weighting accordingly. |
| **User preference** | 10% | Respects saved user preferences (aisle, window, back row, front row). |

### 8.2 Scoring Formula
> **Note:** The overall score is called the **composite score** throughout this section. The variable name `seat_score` in the formula represents the composite score for a single seat.

```
seat_score = (center_score × 0.35)
           + (depth_score × 0.30)
           + (adjacency_bonus × 0.15)
           + (format_adjustment × 0.10)
           + (preference_match × 0.10)
```

Where each sub-score is normalized to 0–100.

### 8.3 Center Score Calculation

> **Note:** `center_col` is the middle column index (e.g., for 20 columns, `center_col = 10`). `max_col` is the total number of columns in the auditorium.

```
center_score = 100 × (1 - |seat_col - center_col| / max_col)
```

### 8.4 Depth Score Calculation

> **Note:** `total_rows` is the count of rows in the auditorium, derived from the number of entries in the `rows` array of the seating layout response. Row IDs are strings (e.g., "A", "B", "J") and must be mapped to sequential indices starting from 1 for the formula.

```
ideal_row_start = total_rows × 0.40
ideal_row_end   = total_rows × 0.70

if row within [ideal_row_start, ideal_row_end]:
    depth_score = 100
else:
    distance_from_zone = min(|row - ideal_row_start|, |row - ideal_row_end|)
    depth_score = max(0, 100 - (distance_from_zone / total_rows) × 200)
```

### 8.5 Adjacency Bonus

> **Note:** All adjacency bonuses are applied **once per booking** (not per seat or per pair). For example, if booking 4 consecutive seats, the +25 bonus applies once, not twice.

- When `ticket_count == 2`: +20 if seats are side-by-side in the same row, +5 if diagonal.
- When `ticket_count >= 3`: +25 if all consecutive in one row, +10 if split across 2 rows with majority together (more than 50% of tickets in one row).
- Non-adjacent selections: −15 penalty.

### 8.6 User Preference Overrides

The integration MUST support these preference flags:

| Preference | Effect |
|---|---|
| `aisle` | Prefer seats adjacent to an aisle; boost center_score by +10 for aisle columns |
| `window` | Prefer seats in the outermost columns |
| `back_row` | Shift depth ideal zone toward the rear (60–100% of rows) |
| `front_row` | Shift depth ideal zone toward the front (10–40% of rows) |
| `no_preference` | Use default scoring |

> **Note:** The `aisle` preference identifies aisle columns by checking if adjacent seat positions have status `Unavailable` (code 2) in the seating layout. The `window` preference is theater-specific: it prefers seats in the outermost columns (closest to the side aisles), not "window" in the airplane sense.

### 8.7 Output

The seat selection engine MUST return the top 3 seat **combinations** ranked by composite score. A "combination" is a set of seats for the requested ticket count. Each combination MUST include:
- Seat labels (e.g., `[J12, J13]`)
- Composite score
- Human-readable rationale (e.g., "Center of auditorium, ideal row depth")

### 8.7.1 Tie-Breaking Policy

When two or more seat combinations have identical composite scores, the following tie-breaking rules apply in order:

1. **Row proximity:** Prefer combinations in rows closer to the screen (lower row letter/number).
2. **Center alignment:** Among tied rows, prefer combinations closer to the horizontal center.
3. **Seat ID ordering:** If still tied, prefer the combination whose lowest seat ID sorts first lexicographically.

This policy ensures deterministic output for the same input.

---

## 9. Testing Strategy

### 9.1 Unit Tests

- **Data model deserialization:** Verify all data model deserialization (TypeScript interfaces per §5.1) correctly parse AMC HAL-style JSON responses, including nested `_embedded` collections.
- **Seat selection edge cases:**
  - Single-seat auditoriums (score = 100 by definition).
  - Fully booked rows (engine returns empty set or falls back to next available row).
  - Auditoriums with irregular layouts (stadium vs. flat, angled rows).
  - Tie-breaking policy is defined in §8.7.1.
- **Format filtering:** Verify `include-attributes` and `exclude-attributes` logic with `and`/`or` operators.
- **Retry logic:** Verify exponential backoff + jitter produces correct delay sequences; verify non-retryable statuses fail immediately.

### 9.2 Contract Tests

- The integration SHOULD include contract tests against the AMC API schema using a tool such as **Schemathesis** or **Pact**.
- Contract tests MUST verify that required response fields (e.g., `performanceNumber`, `layoutId`, `showDateTimeUtc`) are always present and correctly typed.
- Contract tests SHOULD run on each CI pipeline to detect breaking changes in the AMC API before they reach production.

### 9.3 Integration Tests

- Integration tests against the AMC sandbox (if available) MUST cover the full happy path: movie lookup → theater search → showtime query → seat selection → order creation → fulfillment.
- Integration tests SHOULD include failure scenarios: expired orders, sold-out seats, rate-limited responses, and network timeouts.

### 9.4 Observability

The integration MUST emit the following metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `amc_api_requests_total` | Counter | Total API requests by endpoint and status code |
| `amc_api_request_duration_seconds` | Histogram | Latency distribution by endpoint |
| `amc_api_rate_limit_hits_total` | Counter | Number of 429 responses by endpoint (for tuning token bucket) |
| `amc_seat_selection_duration_seconds` | Histogram | Time to score all seats in an auditorium |
| `amc_order_timeout_rate` | Gauge | Fraction of orders that expired before fulfillment |
| `amc_circuit_breaker_state` | Gauge | Current state (closed/open/half-open) per endpoint |

- The integration SHOULD log structured JSON logs including request ID, endpoint, status code, and latency.
- Seat selection latency SHOULD be tracked per auditorium size to detect performance regressions.

---

## 10. Implementation Plan

### Phase 1 — Foundation (Week 1–2)

> **Note:** Phases 1–2 (read-only features) deliver standalone value and MAY be merged/released independently of the e-commerce-dependent Phases 3–5. This allows the team to ship movie discovery and showtime lookup while awaiting `X-AMC-Auth-Token` approval.

- [ ] Register for AMC Developer Portal access
- [ ] Obtain `X-AMC-Vendor-Key`
- [ ] Implement HTTP client wrapper with authentication, rate limiting, and retry logic
- [ ] Build Movie catalog retrieval endpoint
- [ ] Build Theater locator (by name, state, proximity)
- [ ] Write unit tests for data model deserialization

**Exit criteria:** Can list movies and find theaters programmatically.

### Phase 2 — Showtimes (Week 3–4)

- [ ] Implement showtime queries (by theater + date, by movie, by location)
- [ ] Add attribute filtering (IMAX, Dolby Cinema, etc.)
- [ ] Build natural-language query parser ("what's playing this Friday at AMC Downtown?")
- [ ] Implement response formatting for chat interfaces
- [ ] Add optional caching layer for movie/theater data

**Exit criteria:** User can ask "What movies are showing near me?" and get formatted results.

### Phase 3 — Seating Intelligence (Week 5–6)

- [ ] Obtain e-commerce API access (`X-AMC-Auth-Token`)
- [ ] Implement seating layout retrieval
- [ ] Build seat scoring engine with configurable criteria
- [ ] Integrate user preference storage
- [ ] Add seat recommendation to showtime responses

**Exit criteria:** Showtime results include "Best seats: J12, J13 (score: 87/100)".

### Phase 4 — Reservation Booking (Week 7–8)

- [ ] Implement order creation flow
- [ ] Add order status monitoring with expiration alerts
- [ ] Implement order fulfillment submission
- [ ] Build confirmation and barcode retrieval
- [ ] End-to-end testing with real AMC account
- [ ] Error handling for declined payments, expired orders, seat conflicts

**Exit criteria:** User can say "Book two tickets for that showing" and complete the transaction.

### Phase 5 — Polish & Operations (Week 9–10)

> **Note:** The AMC sandbox environment status should be confirmed during Phase 1. If no sandbox is available, integration testing will rely on staging credentials and manual verification.

- [ ] Confirm AMC sandbox/staging availability and obtain test credentials
- [ ] Implement circuit breaker and adaptive throttling
- [ ] Add observability: request logging, error metrics, latency tracking
- [ ] Write integration tests against AMC sandbox (if available)
- [ ] Performance testing under load
- [ ] Documentation and runbooks
- [ ] Production deployment checklist

**Exit criteria:** System is production-ready with monitoring, alerting, and documented operational procedures.

---

## Appendix A: Known Attribute Codes

> **Note:** This list is based on observed AMC API responses and public documentation as of 2026-08. It is NOT exhaustive — AMC may add, remove, or modify attribute codes without notice. The integration SHOULD handle unknown attribute codes gracefully (log a warning, pass through unchanged).

| Code | Name | Applies To |
|------|------|------------|
| `IMAX` | IMAX | Movie, Showtime |
| `IMAX_LASER` | IMAX with Laser | Movie, Showtime |
| `DOLBY_CINEMA` | Dolby Cinema | Movie, Showtime |
| `PRIME` | PRIME Cinema | Movie, Showtime |
| `4DX` | 4DX | Movie, Showtime |
| `SCREENX` | ScreenX | Movie, Showtime |
| `LASED` | Digital (Laser) | Showtime |
| `3D` | 3D | Movie, Showtime |
| `ADULTS_ONLY` | Rated R / NC-17 | Movie |
| `PREMIERE` | Premiere Showing | Showtime |
| `MATINEE` | Matinee Pricing | Showtime |
| `VIP_LOUNGE` | VIP Experience | Theatre |

## Appendix B: HTTP Headers Reference

| Header | Value | Required For |
|--------|-------|-------------|
| `X-AMC-Vendor-Key` | `{api-key}` | All endpoints |
| `X-AMC-Auth-Token` | `{auth-token}` | Seating v3, Order v3, Barcode v3 |
| `Accept` | `application/json` | Recommended on all requests |
| `Content-Type` | `application/json` | POST/PUT requests |

## Appendix C: Glossary

| Term | Definition |
|------|-----------|
| **Performance Number** | AMC's internal identifier for a specific showing of a movie at a theater; required for seating and order APIs |
| **Layout ID** | Identifier for the physical seat arrangement of an auditorium; paired with performance number to query availability |
| **HAL Envelope** | Hypertext Application Language response format used by AMC; collections include `_embedded` data and `_links` navigation |
| **Embargoed** | Showtimes for movies not yet publicly announced (e.g., advance industry screenings); see §4.4.2 |
| **WAMC** | "When At My Cinema" — AMC's internal session identifier for a showtime (not exposed by the public API; retained for reference only) |

---

*End of specification.*
