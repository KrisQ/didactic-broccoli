package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/KrisQ/didactic-broccoli/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type chirpResponse struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	message := fmt.Sprintf(`
			<html>
				<body>
					<h1>Welcome, Chirpy Admin</h1>
					<p>Chirpy has been visited %d times!</p>
				</body>
			</html>
		`, cfg.fileserverHits.Load())
	_, err := w.Write([]byte(message))
	if err != nil {
		fmt.Println("oh well")
	}
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	cfg.fileserverHits.Store(0)
	message := fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())
	err := cfg.dbQueries.DeleteUsers(r.Context())
	if err != nil {
		fmt.Println("oh well")
		return
	}
	err = cfg.dbQueries.DeleteChrips(r.Context())
	if err != nil {
		fmt.Println("oh well")
		return
	}
	_, err = w.Write([]byte(message))
	if err != nil {
		fmt.Println("oh well")
		return
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	type reqVals struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	decoder := json.NewDecoder(r.Body)
	params := reqVals{}
	w.Header().Set("Content-Type", "application/json")
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("couldn't decode chirp")
		w.WriteHeader(500)
		return
	} else if len(params.Body) > 140 {
		log.Printf("chirp too long")
		w.WriteHeader(400)
		return
	} else {
		params.Body = removeProfanity(params.Body)
	}
	chirp, err := cfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   params.Body,
		UserID: params.UserID,
	})
	if err != nil {
		log.Printf("couldn't create chirp")
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(201)
	response := chirpResponse{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}
	dat, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	_, err = w.Write(dat)
	if err != nil {
		log.Printf("Error writing %s", err)
		w.WriteHeader(500)
		return
	}
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type reqVals struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	params := reqVals{}
	err := decoder.Decode(&params)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Error writing %s", err)
		w.WriteHeader(500)
		return
	}
	user, err := cfg.dbQueries.CreateUser(r.Context(), params.Email)
	if err != nil {
		log.Printf("Error creating user %s", err)
		w.WriteHeader(500)
		return
	}
	response := userResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	dat, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshilling res %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(201)
	_, err = w.Write(dat)
	if err != nil {
		log.Printf("Error writing %s", err)
		w.WriteHeader(500)
		return
	}
}

func main() {
	const port = "8080"

	if err := godotenv.Load(); err != nil {
		log.Printf("godotenv.Load error: %v", err)
	}

	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("sql.Open error: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("db.Ping error: %v", err)
	}

	var apiCfg apiConfig
	apiCfg.dbQueries = database.New(db)
	apiCfg.platform = os.Getenv("PLATFORM")

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	mux.Handle("GET /api/healthz", http.HandlerFunc(healthHandler))

	mux.Handle("GET /api/chirps", http.HandlerFunc(apiCfg.getChirps))
	mux.Handle("POST /api/chirps", http.HandlerFunc(apiCfg.createChirp))

	mux.Handle("POST /api/users", http.HandlerFunc(apiCfg.createUser))

	mux.Handle("GET /admin/metrics", http.HandlerFunc(apiCfg.metricsHandler))

	mux.Handle("POST /admin/reset", http.HandlerFunc(apiCfg.resetHandler))

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}
