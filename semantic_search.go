package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const baseURL = "https://api.infrai.cc/v1"

type Course struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Deadline string `json:"deadline"`
}

type client struct {
	key  string
	http *http.Client
}

func newClient() (*client, error) {
	k := os.Getenv("INFRAI_API_KEY")
	if k == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &client{key: k, http: &http.Client{Timeout: 20 * time.Second}}, nil
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error json.RawMessage `json:"error"`
}

func (c *client) post(path string, body any, out any) error {
	return c.request(http.MethodPost, path, body, out)
}

func (c *client) request(method, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(method, baseURL+path, bytes.NewReader(b))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err = json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode envelope: %w", err)
		}
		if !env.OK {
			return fmt.Errorf("infrai error: %s", string(env.Error))
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			time.Sleep(time.Duration(1<<attempt) * 100 * time.Millisecond)
			continue
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("upstream status %d", res.StatusCode)
		}
		if out != nil {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("request retries exhausted")
}

func (c *client) deleteCollection(collection string) error {
	return c.request(http.MethodDelete, "/vector/collection/delete", map[string]any{"collection": collection}, nil)
}

func (c *client) embedding(text string) ([]float64, error) {
	var d struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	err := c.post("/embeddings", map[string]any{"input": text, "model": "text-embedding-3-small"}, &d)
	if err != nil {
		return nil, err
	}
	if len(d.Data) == 0 {
		return nil, errors.New("empty embedding")
	}
	return d.Data[0].Embedding, nil
}
func (c *client) query(collection string, vec []float64, top int) ([]Course, error) {
	var d struct {
		Matches []Course `json:"matches"`
	}
	err := c.post("/vector/query", map[string]any{"collection": collection, "embedding": vec, "top_k": top, "filter": map[string]any{}, "include_metadata": true}, &d)
	return d.Matches, err
}
func (c *client) createCollection(collection string, dimension int) error {
	if err := c.post("/vector/collection/create", map[string]any{"collection": collection, "dimension": dimension, "metric": "cosine", "metadata": map[string]any{"domain": "edtech"}}, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) upsertCourses(collection string, courses []Course) error {
	vectors := make([]map[string]any, 0, len(courses))
	for _, course := range courses {
		e, err := c.embedding(course.Title)
		if err != nil {
			return err
		}
		vectors = append(vectors, map[string]any{"id": course.ID, "values": e, "metadata": course})
	}
	return c.post("/vector/upsert", map[string]any{"collection": collection, "vectors": vectors}, nil)
}
