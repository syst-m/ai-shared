# AMC Theatres Public API Reference

**Last Updated:** 2026-09-03  
**Base URL:** `https://api.amctheatres.com`  
**API Key Version:** 2 (current production)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Authentication](#2-authentication)
3. [Rate Limiting](#3-rate-limiting)
4. [Error Format](#4-error-format)
5. [Pagination](#5-pagination)
6. [HAL Envelope](#6-hal-envelope)
7. [Working Endpoints](#7-working-endpoints)
   - [7.1 Movies — List / Search](#71-movies--list--search)
   - [7.2 Movies — Detail](#72-movies--detail)
   - [7.3 Theatres — List](#73-theatres--list)
   - [7.4 Theatres — Detail](#74-theatres--detail)
   - [7.5 Showtimes — Theatre-specific](#75-showtimes--theatre-specific)
   - [7.6 Showtimes — Theatre-specific by Date](#76-showtimes--theatre-specific-by-date)
   - [7.7 Showtimes — Earliest Showing](#77-showtimes--earliest-showing)
   - [7.8 Showtimes — Single Showtime](#78-showtimes--single-showtime)
   - [7.9 Showtimes — Current Location View](#79-showtimes--current-location-view)
   - [7.10 Cast & Crew (v1)](#710-cast--crew-v1)
8. [Retired / Dead Endpoints](#8-retired--dead-endpoints)
9. [Gotchas & Known Issues](#9-gotchas--known-issues)
10. [Attribute Codes](#10-attribute-codes)
11. [Response Field Reference](#11-response-field-reference)

---

## 1. Overview

The AMC Theatres public API is a RESTful JSON API accessed via `https://api.amctheatres.com`. All read endpoints require a vendor API key. Responses use a HAL (Hypertext Application Language) envelope with `_embedded` collections and `_links` navigation.

As of **2026-08-30**, AMC migrated to a new API backend. Several previously-documented endpoints now return **404 Not Found**. This document reflects the **live API as of 2026-09-03**.

---

## 2. Authentication

All requests require the `X-AMC-Vendor-Key` header:

```
GET /v2/movies?name=*
Host: api.amctheatres.com
Accept: application/json
X-AMC-Vendor-Key: {your-api-key}
```

| Header | Required | Description |
|---|---|---|
| `X-AMC-Vendor-Key` | Yes | Read-only vendor key obtained from the [AMC Developer Portal](https://developers.amctheatres.com/GettingStarted/NewVendorRequest) |
| `Accept` | Recommended | `application/json` |

**Auth failure behavior:** An invalid or missing key returns **HTTP 404 with an empty body** — not a 401. An empty key string returns **HTTP 400** with a JSON error body.

---

## 3. Rate Limiting

- AMC uses a credit-based rate limiting system.
- The API **does not** return rate-limit headers (`X-RateLimit-*`, `Retry-After`) in normal responses.
- When rate-limited, the API returns **HTTP 429**. The `Retry-After` header, if present, MUST be respected.
- No documented per-endpoint credit costs are publicly available.
- **Recommendation:** Implement client-side token-bucket throttling (e.g., 1 token per 2 seconds, capacity 10).

---

## 4. Error Format

Errors follow a HAL-style `errors` array:

```json
{
  "errors": [
    {
      "id": "5f7db9d6-ed72-4d70-b866-fe9b51336cb2",
      "code": 5308,
      "exceptionMessage": "Search criteria required."
    }
  ]
}
```

| Field | Type | Description |
|---|---|---|
| `errors` | array | Array of error objects |
| `errors[].id` | string | UUID for the specific error instance |
| `errors[].code` | integer | Machine-readable error code |
| `errors[].exceptionMessage` | string | Human-readable message |

**Common error codes:**

| Code | HTTP Status | Meaning |
|---|---|---|
| `1` | 400 | The request requires vendor authentication |
| `12004` | 404 | No matching Web application endpoint was found |
| `5308` | 400 | Search criteria required |

---

## 5. Pagination

All collection endpoints support pagination via query parameters:

| Parameter | Type | Default | Max | Description |
|---|---|---|---|---|
| `page-number` | integer | 1 | — | Page number (1-based) |
| `page-size` | integer | 10 | 1000 | Results per page |

**Response envelope fields:**

| Field | Type | Description |
|---|---|---|
| `pageSize` | integer | Requested page size |
| `pageNumber` | integer | Current page number |
| `count` | integer | Total number of items across all pages |
| `lastUpdatedDateUtc` | string (ISO 8601) | Last update timestamp (showtimes endpoints only) |

**Navigation:** Use the `_links.next.href` absolute URL to fetch the next page. The `next` link is present only when more pages exist.

---

## 6. HAL Envelope

All responses wrap content in a HAL envelope:

```json
{
  "pageSize": 10,
  "pageNumber": 1,
  "count": 523,
  "_links": {
    "self": { "href": "https://api.amctheatres.com/v2/theatres?page-number=1&page-size=10" },
    "next": { "href": "https://api.amctheatres.com/v2/theatres?page-number=2&page-size=10" }
  },
  "_embedded": {
    "theatres": [ /* array of theatre objects */ ]
  }
}
```

| Field | Type | Description |
|---|---|---|
| `_links` | object | Navigation links. Always includes `self`. Includes `next` when more pages exist. |
| `_embedded` | object | Collection objects keyed by resource type (`theatres`, `movies`, `showtimes`). |

**Important:** `_links.next.href` is an **absolute URL** (not relative). When following pagination, use the full URL directly — do not prepend the base URL.

---

## 7. Working Endpoints

### 7.1 Movies — List / Search

```
GET /v2/movies
```

**Query Parameters:**

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `name` | string | **Yes** | — | Search term for movie name. Use `*` for a wildcard catalog listing. |
| `movie-status` | string | No | `currently-playing` | Filter: `currently-playing`, `upcoming`, or `all` |
| `page-number` | integer | No | 1 | Page number |
| `page-size` | integer | No | 10 | Results per page (max 1000) |

**Response:** HAL envelope with `_embedded.movies` array.

**Example Request:**
```
GET /v2/movies?name=Odyssey&page-size=5
X-AMC-Vendor-Key: {key}
```

**Example Response (first item in collection):**
```json
{
  "id": 53191,
  "name": "The Odyssey",
  "sortableName": "Odyssey, The",
  "starringActors": "Anne Hathaway, Matt Damon, Tom Holland",
  "directors": "Christopher Nolan",
  "mpaaRating": "R",
  "score": 0.15375,
  "slug": "the-odyssey-53191",
  "hasScheduledShowtimes": false,
  "websiteUrl": "https://www.amctheatres.com/movies/the-odyssey-53191",
  "showtimesUrl": "https://www.amctheatres.com/movies/the-odyssey-53191/showtimes",
  "availableForAList": true,
  "preferredMediaType": "Theatrical",
  "attributes": [],
  "media": {
    "posterThumbnail": "",
    "posterStandard": "",
    "trailerHd": "https://..."
  },
  "_links": {
    "self": { "href": "https://api.amctheatres.com/v2/movies/53191" }
  }
}
```

**Status Codes:**
| Code | Meaning |
|---|---|
| 200 | Success |
| 400 | Missing `name` parameter |
| 404 | Invalid / missing API key |

---

### 7.2 Movies — Detail

```
GET /v2/movies/{movie-id}
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `movie-id` | integer | AMC movie ID |

**Response fields (additional to list items):**

| Field | Type | Description |
|---|---|---|
| `synopsis` | string | Film synopsis |
| `runTime` | integer | Duration in minutes |
| `genre` | string | Genre string |
| `releaseDateUtc` | string (ISO 8601) | Release date |
| `earliestShowingUtc` | string (ISO 8601) | Earliest showing date |
| `onlineTicketAvailabilityDateUtc` | string (ISO 8601) | When tickets go on sale |
| `distributorId` | integer | Distributor ID |
| `distributorCode` | string | Distributor code |
| `wwmReleaseNumber` | integer | Internal release number |
| `vuduUrl` | string | Vudu purchase link (may be empty) |
| `attributes` | array | Format attributes (IMAX, 4DX, etc.) |
| `media` | object | Poster/thumbnail URLs |

**Status Codes:**
| Code | Meaning |
|---|---|
| 200 | Success |

---

### 7.3 Theatres — List

```
GET /v2/theatres
```

**Query Parameters:**

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `name` | string | No | — | Filter by theatre name (substring match) |
| `page-number` | integer | No | 1 | Page number |
| `page-size` | integer | No | 10 | Results per page (max 1000) |

**Response:** HAL envelope with `_embedded.theatres` array.

**Example Response (first item):**
```json
{
  "id": 6,
  "longName": "AMC Esquire 7",
  "name": "AMC Esquire 7",
  "slug": "amc-esquire-7",
  "brand": "AMC",
  "ticketable": "AMC",
  "isClosed": false,
  "guestServicesPhoneNumber": "3147813300",
  "utcOffset": "-05:00",
  "timezone": "CENTRAL TIME",
  "timezoneAbbreviation": "CDT",
  "websiteUrl": "https://www.amctheatres.com/movie-theatres/st-louis/amc-esquire-7",
  "facebookUrl": "https://www.facebook.com/AMCEsquire7",
  "deliveryToSeat": false,
  "onlineConcessions": true,
  "hasMultipleKitchens": false,
  "subscriptionUsageLevel": 3000,
  "loyaltyVersionId": 3,
  "westWorldMediaNumber": 5054,
  "concessionsDeliveryOptions": ["EXPRESSPICKUP"],
  "convenienceFeeTaxPercent": 9.488,
  "convenienceFeeTaxFlatAmount": 0.0,
  "redemptionMethods": ["Credit Card Swipe"],
  "closures": [
    {
      "startDateTimeUtc": "2020-03-16T05:00:00.000Z",
      "endDateTimeUtc": "2020-08-27T04:59:59.000Z"
    }
  ],
  "estimatedConcessionsOrderFees": [
    { "name": "service", "cost": 1.99, "quantity": 1 }
  ],
  "location": {
    "addressLine1": "6706 Clayton Road",
    "addressLine2": "",
    "city": "SAINT LOUIS",
    "cityUrlSuffixText": "saint-louis",
    "postalCode": "63117-1604",
    "state": "MO",
    "stateName": "MISSOURI",
    "stateUrlSuffixText": "missouri",
    "country": "United States",
    "latitude": 38.634182,
    "longitude": -90.316645,
    "directionsUrl": "http://bing.com/maps/...",
    "marketName": "St. Louis",
    "marketUrlSuffixText": "st-louis",
    "marketId": 106
  },
  "attributes": [
    {
      "code": "macguffins",
      "name": "MacGuffins Bar",
      "description": "Enjoy a beer or wine with your movie..."
    },
    {
      "code": "reservedseating",
      "name": "Reserved Seating",
      "description": "Select your seat when you buy your tickets..."
    }
  ],
  "media": {
    "theatreImageIcon": "https://...",
    "theatreImageStandard": "https://...",
    "theatreImageLarge": "https://..."
  },
  "_links": {
    "self": { "href": "https://api.amctheatres.com/v2/theatres/6" }
  }
}
```

**Status Codes:**
| Code | Meaning |
|---|---|
| 200 | Success |

---

### 7.4 Theatres — Detail

```
GET /v2/theatres/{theatre-id}
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `theatre-id` | integer | AMC theatre ID (NOT the old "theatre number") |

**Response:** Same structure as the list endpoint, but for a single theatre.

**Key fields:**
| Field | Type | Description |
|---|---|---|
| `id` | integer | Theatre ID (primary key) |
| `name` | string | Display name |
| `longName` | string | Full name (same as `name` in most cases) |
| `slug` | string | URL-friendly slug |
| `location.latitude` | float | Latitude |
| `location.longitude` | float | Longitude |
| `location.city` | string | City (uppercase) |
| `location.state` | string | State code (uppercase) |
| `location.postalCode` | string | ZIP code |
| `location.marketName` | string | Market name |
| `location.marketId` | integer | Market ID |
| `guestServicesPhoneNumber` | string | Phone number (no formatting) |
| `attributes` | array | Theatre-level attributes (formats, amenities) |

**⚠️ No `number` field:** The response does **not** contain a `number` field. Use `id` as the primary identifier.

**Status Codes:**
| Code | Meaning |
|---|---|
| 200 | Success |

---

### 7.5 Showtimes — Theatre-specific

```
GET /v2/theatres/{theatre-id}/showtimes
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `theatre-id` | integer | AMC theatre ID |

**Query Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `page-number` | integer | Page number (default 1) |
| `page-size` | integer | Results per page (default 10, max 1000) |

**Response:** HAL envelope with `_embedded.showtimes` array. **No `theatres` embedded section.**

**Status Codes:**
| Code | Meaning |
|---|---|
| 200 | Success |

---

### 7.6 Showtimes — Theatre-specific by Date

```
GET /v2/theatres/{theatre-id}/showtimes/{date}
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `theatre-id` | integer | AMC theatre ID |
| `date` | string | **MM-DD-YYYY** format (NOT ISO 8601) |

**Query Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `page-number` | integer | Page number (default 1) |
| `page-size` | integer | Results per page (default 10, max 1000) |
| `movie-id` | integer | Filter to a specific movie |
| `include-attributes` | string | Comma-delimited attribute codes to include (e.g., `IMAX`) |
| `exclude-attributes` | string | Comma-delimited attribute codes to exclude |
| `attribute-operator` | string | **`or`** only — `and` returns 404 |

**Example:**
```
GET /v2/theatres/2325/showtimes/09-03-2026?movie-id=76238&include-attributes=IMAX
```

**Status Codes:**
| Code | Meaning |
|---|---|
| 200 | Success |

---

### 7.7 Showtimes — Earliest Showing

```
GET /v2/theatres/{theatre-id}/movies/{movie-id}/earliest-showtime
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `theatre-id` | integer | AMC theatre ID |
| `movie-id` | integer | AMC movie ID |

**Response:** Single showtime object (not wrapped in HAL envelope).

**Status Codes:**
| Code | Meaning |
|---|---|
| 200 | Success |

---

### 7.8 Showtimes — Single Showtime

```
GET /v2/showtimes/{showtime-id}
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `showtime-id` | integer | Showtime ID |

**Response:** Single showtime object (not wrapped in HAL envelope).

**Status Codes:**
| Code | Meaning |
|---|---|
| 200 | Success |

---

### 7.9 Showtimes — Current Location View

```
GET /v2/showtimes/views/current-location/{date}/{latitude}/{longitude}
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `date` | string | **MM-DD-YYYY** format (NOT ISO 8601) |
| `latitude` | float | Latitude (e.g., `37.784`) |
| `longitude` | float | Longitude (e.g., `-122.404`) |

**Query Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `page-number` | integer | Page number (default 1) |
| `page-size` | integer | Results per page (default 10) |
| `movie-id` | integer | Filter to a specific movie |
| `include-attributes` | string | Include only showtimes with these attributes |
| `exclude-attributes` | string | Exclude showtimes with these attributes |
| `attribute-operator` | string | **`or`** only |
| `theatre-id` | integer | ⚠️ **Ignored** — does NOT filter server-side |

**⚠️ Important:** The `theatre-id` query parameter is **accepted but ignored**. The endpoint returns showtimes for ALL theatres near the coordinates. Client-side filtering by `theatreId` is required.

**Response:** HAL envelope with `_embedded.showtimes` array. The `count` field reflects the total across all nearby theatres.

**Example:**
```
GET /v2/showtimes/views/current-location/09-03-2026/37.784/-122.404
```

**Status Codes:**
| Code | Meaning |
|---|---|
| 200 | Success |

---

### 7.10 Cast & Crew (v1)

```
GET /v1/movie/{movie-id}/cast-crew
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `movie-id` | integer | AMC movie ID |

**Status:** Still working (v1 endpoint). No documented v1 spec.

**Status Codes:**
| Code | Meaning |
|---|---|
| 200 | Success |

---

## 8. Retired / Dead Endpoints

The following endpoints returned **404 Not Found** as of 2026-09-03:

| Endpoint | Method | Previous Purpose |
|---|---|---|
| `/v2/showtimes` | GET | (No path — never documented) |
| `/v2/showtimes/views` | GET | (No path — never documented) |
| `/v2/showtimes?theatre-id=X` | GET | Theatre-based showtimes query |
| `/v2/showtimes?theatre-id=X&date=Y` | GET | Date-filtered showtimes query |
| `/v2/locations/name/{name}` | GET | Theatre search by name |
| `/v2/locations/state/{state}` | GET | Theatre search by state |
| `/v2/theatres/{id}/movies` | GET | Movies playing at a theatre |
| `/v2/media/movies/{id}/{type}` | GET | Movie media (poster, trailer) |
| `/v2/media/theatres/{id}/{type}` | GET | Theatre media (images) |
| `/v2/seating-layouts/{theatre}/{performance}` | GET | Seating layout |
| `/v3/seating-layouts/{theatre}/{performance}` | GET | Seating layout (v3) |
| `/v3/orders` | GET | Orders list |
| `/v1/theatres` | GET | Theatre list (v1) |

**Migration guidance:**
- Theatre search by name → Use `GET /v2/theatres?name={query}` instead.
- Theatre search by state → No direct replacement; list all theatres and filter client-side.
- Per-theatre showtimes → Use `GET /v2/theatres/{id}/showtimes` or `GET /v2/theatres/{id}/showtimes/{date}`.
- Location-based showtimes → Use `GET /v2/showtimes/views/current-location/{date}/{lat}/{lon}`.
- Seating layouts → No replacement available with vendor key alone.

---

## 9. Gotchas & Known Issues

### 9.1 Theatre Renumbering

AMC renumbered its theatre IDs around 2026. The old "AMC Metreon 8" (San Francisco) is now **theatre ID 2325**. The old number `8` now maps to a Kansas City theatre (with empty name and zeroed coordinates).

| Old Number | New ID | Name | City |
|---|---|---|---|
| 8 | 2325 | AMC Metreon 16 | SAN FRANCISCO |
| 8 | 8 | (empty) | KANSAS CITY |

**Always use `id` from the API response, never hard-coded numbers.**

### 9.2 Date Format

The date parameter in showtime paths uses **MM-DD-YYYY** (e.g., `09-03-2026`), **NOT** ISO 8601 (`2026-09-03`). Using ISO format returns 404.

### 9.3 No `number` Field

Theatre responses **do not** contain a `number` field. The primary identifier is `id`. Code that expects a `number` field will fail.

### 9.4 Client-Side Theatre Filtering

The `theatre-id` query parameter on the current-location showtimes view is **accepted but ignored**. All nearby theatre showtimes are returned; filter client-side by the `theatreId` field.

### 9.5 Auth Returns 404

An invalid or missing vendor key returns **HTTP 404 with an empty body**, not 401. Detect auth failures by checking if the response body is empty on a normally-successful endpoint.

### 9.6 `attribute-operator=and` Returns 404

The `and` operator for multi-attribute filtering returns 404. Only `or` is supported.

### 9.7 Movie Search Requires `name`

The `/v2/movies` endpoint **requires** the `name` query parameter. Omitting it returns HTTP 400 with error code 5308. Use `name=*` for a full catalog listing.

### 9.8 HAL `_links.next` Are Absolute URLs

The `_links.next.href` value is a full absolute URL (e.g., `https://api.amctheatres.com/v2/movies?name=*&page-number=2&page-size=10`). Use it directly — do not prepend the base URL.

### 9.9 Movie Search is Case-Insensitive with Fuzzy Matching

The `name` parameter on `/v2/movies` performs fuzzy matching. Searching for "Avatar" returns "TOSCA (AVATAR FILMS)" as the first result. Results are sorted by relevance score.

### 9.10 Rate Limit Headers Absent

The API does not include `X-RateLimit-*` headers in responses. Rate limit state must be inferred from 429 responses.

---

## 10. Attribute Codes

Observed attribute codes from live API responses:

| Code | Name | Applies To |
|---|---|---|
| `70MM` | 70mm | Showtime |
| `AMCARTISANFILMS` | AMC Artisan Films | Showtime |
| `AMCCLUBROCKERS` | AMC Club Rockers | Showtime |
| `AONELIVE` | AONe Live | Showtime |
| `CLOSEDCAPTION` | Closed Caption | Theatre |
| `DESCRIPTIVEVIDEO` | Audio Description | Theatre |
| `DISNEYRWDS` | Disney RWDS | Showtime |
| `IMAX` | IMAX at AMC | Showtime |
| `IMAX70MM` | IMAX 70MM | Showtime |
| `LASERATAMC` | Laser at AMC | Showtime |
| `macguffins` | MacGuffins Bar | Theatre |
| `digitalprojection` | Digital Projection | Theatre |
| `reservedseating` | Reserved Seating | Theatre |
| `theatrerentals` | Theatre Rentals | Theatre |
| `assistedlisteningdevices` | Assisted Listening Devices | Theatre |
| `wheelchairaccess` | Wheelchair Access | Theatre |
| `nooutsidefoodandbeverage` | No Outside Food and Beverage | Theatre |
| `alcoholcardingpolicy` | Alcohol Carding Policy | Theatre |
| `REALD3D` | RealD 3D | Showtime |
| `RECLINERSEATING` | Recliner Seating | Theatre |
| `THRLSCHLS` | Thrillschls | Showtime |

---

## 11. Response Field Reference

### 11.1 Showtime Object

| Field | Type | Description |
|---|---|---|
| `id` | integer | Showtime ID |
| `internalReleaseNumber` | integer | Internal release number |
| `performanceNumber` | integer | Performance number (required for seating/order APIs) |
| `movieId` | integer | Movie ID |
| `movieName` | string | Movie title |
| `sortableMovieName` | string | Sortable title |
| `genre` | string | Genre (uppercase) |
| `showDateTimeUtc` | string (ISO 8601) | UTC show time |
| `showDateTimeLocal` | string (ISO 8601, no tz) | Local show time |
| `sellUntilDateTimeUtc` | string (ISO 8601) | When sales close |
| `isSoldOut` | boolean | Sold out |
| `isAlmostSoldOut` | boolean | Nearly sold out |
| `isCanceled` | boolean | Canceled |
| `utcOffset` | string | UTC offset (e.g., `-07:00`) |
| `theatreId` | integer | Theatre ID |
| `auditorium` | integer | Auditorium number |
| `layoutId` | integer | Layout ID |
| `layoutVersionNumber` | integer | Layout version |
| `runTime` | integer | Duration in minutes |
| `mpaaRating` | string | MPAA rating |
| `premiumFormat` | string | Premium format name (e.g., "70mm") |
| `purchaseUrl` | string | Purchase URL |
| `mobilePurchaseUrl` | string | Mobile purchase URL |
| `movieUrl` | string | Movie page URL |
| `wwmReleaseNumber` | integer | WWM release number |
| `lastUpdatedDateUtc` | string (ISO 8601) | Last update time |
| `isDiscountMatineePriced` | boolean | Matinee pricing |
| `discountMatineeMessage` | string | Matinee discount message |
| `visibilityDateTimeUtc` | string (ISO 8601) | When showtime became visible |
| `isDiscountDaysEligible` | boolean | Discount day eligible |
| `hasTrailers` | boolean | Has trailers |
| `inTheatreTicketingOnly` | boolean | In-theatre only |
| `estimatedFees` | array | Estimated additional fees |
| `attributes` | array | Format/attribute objects |
| `ticketPrices` | array | Pricing objects |
| `media` | object | Movie media URLs |
| `languages` | object | Available languages |

### 11.2 Ticket Price Object

| Field | Type | Description |
|---|---|---|
| `price` | float | Price in USD |
| `type` | string | Price type (ADULT, CHILD, SENIOR) |
| `sku` | string | SKU identifier |
| `tax` | float | Tax amount |
| `priceCode` | string | Price code |
| `posType` | string | POS type |
| `agePolicy` | string | Age restriction (optional) |

### 11.3 Estimated Fee Object

| Field | Type | Description |
|---|---|---|
| `name` | string | Fee name (e.g., "convenience") |
| `cost` | float | Fee amount |
| `tax` | float | Tax on fee |
| `quantity` | integer | Quantity |

### 11.4 Attribute Object

| Field | Type | Description |
|---|---|---|
| `code` | string | Attribute code |
| `name` | string | Display name |
| `description` | string | Full description |

---

*End of reference. Last verified against live API: 2026-09-03.*
