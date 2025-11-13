# API Testing Guide

This document provides all the endpoints and test cases for the Chirpy API.

## Base URL
```
http://localhost:8080
```

## Endpoints

### 1. Health Check
**GET** `/api/healthz`

Returns a simple health check status.

**Request:**
```bash
curl http://localhost:8080/api/healthz
```

**Expected Response:**
- Status: `200 OK`
- Body: `OK`
- Content-Type: `text/plain; charset=utf-8`

---

### 2. Validate Chirp
**POST** `/api/validate_chirp`

Validates a chirp message. Chirps must be 140 characters or less. Profanity words (kerfuffle, sharbert, fornax) are automatically replaced with `****`.

**Request:**
```bash
curl -X POST http://localhost:8080/api/validate_chirp \
  -H "Content-Type: application/json" \
  -d '{"body": "This is a valid chirp message!"}' | jq
```

**Request Body:**
```json
{
  "body": "string"
}
```

**Success Response (200):**
```json
{
  "cleaned_body": "This is a valid chirp message!"
}
```

**Error Response - Too Long (400):**
```json
{
  "error": "Chirp is too long"
}
```

**Error Response - Server Error (500):**
```json
{
  "error": "Something went wrong"
}
```

#### Test Cases

**Valid Chirp:**
```bash
curl -X POST http://localhost:8080/api/validate_chirp \
  -H "Content-Type: application/json" \
  -d '{"body": "This is a valid chirp message!"}' | jq
```

**Chirp Too Long (>140 characters):**
```bash
curl -X POST http://localhost:8080/api/validate_chirp \
  -H "Content-Type: application/json" \
  -d '{"body": "This is a very long chirp message that exceeds the 140 character limit and should return an error because it is way too long to be a valid chirp!"}' | jq
```

**Chirp with Profanity:**
```bash
curl -X POST http://localhost:8080/api/validate_chirp \
  -H "Content-Type: application/json" \
  -d '{"body": "What a kerfuffle this is! And sharbert too!"}' | jq
```

Expected cleaned body: `"What a **** this is! And **** too!"`

**Profanity words to test:**
- `kerfuffle` → `****`
- `sharbert` → `****`
- `fornax` → `****`

---

### 3. Get Metrics
**GET** `/admin/metrics`

Returns an HTML page showing the number of times the file server has been accessed.

**Request:**
```bash
curl http://localhost:8080/admin/metrics
```

**Expected Response:**
- Status: `200 OK`
- Content-Type: `text/html; charset=utf-8`
- Body: HTML page with hit count

**Example Response:**
```html
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited 5 times!</p>
  </body>
</html>
```

---

### 4. Reset Metrics
**POST** `/admin/reset`

Resets the file server hit counter to 0.

**Request:**
```bash
curl -X POST http://localhost:8080/admin/reset
```

**Expected Response:**
- Status: `200 OK`
- Content-Type: `text/plain; charset=utf-8`
- Body: `Hits: 0`

---

### 5. Static File Server
**GET** `/app/*`

Serves static files from the project directory. Each request increments the metrics counter.

**Request:**
```bash
curl http://localhost:8080/app/index.html
```

**Expected Response:**
- Status: `200 OK`
- Body: File contents
- Note: Each request increments the metrics counter

---

## Testing Workflow

1. **Start the server:**
   ```bash
   go run main.go utils.go
   ```

2. **Check health:**
   ```bash
   curl http://localhost:8080/api/healthz
   ```

3. **Test chirp validation:**
   - Test valid chirp
   - Test chirp that's too long
   - Test profanity filtering

4. **Check metrics:**
   ```bash
   curl http://localhost:8080/admin/metrics
   ```

5. **Access static files (increments counter):**
   ```bash
   curl http://localhost:8080/app/index.html
   ```

6. **Check metrics again (should be higher):**
   ```bash
   curl http://localhost:8080/admin/metrics
   ```

7. **Reset metrics:**
   ```bash
   curl -X POST http://localhost:8080/admin/reset
   ```

8. **Verify reset:**
   ```bash
   curl http://localhost:8080/admin/metrics
   ```

---

## Using with Bruno/Postman/Insomnia

### Collection Import
You can manually create requests in your API client using the information above, or use the `bruno-collection` folder if you're using Bruno.

### Environment Variables
Create an environment with:
- `base_url`: `http://localhost:8080`

Then use `{{base_url}}/api/healthz` etc. in your requests.

---

## Notes

- The server runs on port `8080` by default
- All endpoints are case-sensitive
- The `/app/*` endpoint serves files from the current directory
- Metrics are stored in memory and reset when the server restarts
- Profanity filtering is case-insensitive

