package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
    "encoding/json"
    "strings"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}

	type chirpReq struct {
		Body string `json:"body"`
	}

	type chirpResponse struct {
		CleanedBody string `json:"cleaned_body"`  // Cleaned body to return
	}

	type chirpErr struct {
		Error string `json:"error"`
	}

	var params chirpReq
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(chirpErr{Error: "Something went wrong"})
		return
	}

	if len(params.Body) > 140 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(chirpErr{Error: "Chirp is too long"})
		return
	}

	words := strings.Fields(params.Body)
	for i, word := range words {
		cleanWord := strings.Trim(word, "!.,?")

		for _, profane := range profaneWords {
			if strings.EqualFold(cleanWord, profane) {
				words[i] = strings.Replace(word, cleanWord, "****", 1)
			}
		}
	}

	cleanedBody := strings.Join(words, " ")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(chirpResponse{
		CleanedBody: cleanedBody,
	})
}

func main() {
	const filepathRoot = "."
	const port = "8080"

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}

	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))))
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
    mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("<html> <body> <h1>Welcome, Chirpy Admin</h1> <p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
