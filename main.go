package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

var courses = []Course{{ID: "algebra-quiz", Title: "Algebra quiz and worked examples", Deadline: "2026-09-20"}, {ID: "study-skills", Title: "Study skills for remote learners", Deadline: "2026-09-28"}}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	c, err := newClient()
	if err != nil {
		return err
	}
	collection := fmt.Sprintf("edtech-courses-%d", time.Now().UnixNano())
	if err := c.createCollection(collection, 1536); err != nil {
		return err
	}
	defer func() {
		if cleanupErr := c.deleteCollection(collection); cleanupErr != nil {
			err = errors.Join(err, cleanupErr)
		}
	}()
	if err := c.upsertCourses(collection, courses); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query().Get("q")
		if q == "" {
			http.Error(w, "q is required", http.StatusBadRequest)
			return
		}
		vec, err := c.embedding(q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		found, err := c.query(collection, vec, 5)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"query": q, "courses": orderByDeadline(found)})
	})
	server := &http.Server{Addr: ":8080", Handler: mux}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()
	log.Println("semantic search listening on :8080")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
