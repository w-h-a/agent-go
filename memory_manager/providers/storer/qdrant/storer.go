package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/w-h-a/agent/memory_manager/providers/storer"
	getsafe "github.com/w-h-a/agent/util/get_safe"
)

type qdrantStorer struct {
	options storer.Options
	client  *http.Client
}

func (s *qdrantStorer) Store(ctx context.Context, spaceId string, sessionId string, content string, metadata map[string]any, vector []float32) error {
	id := uuid.New().String()

	payload := map[string]any{
		"session_id": sessionId,
		"content":    content,
		"metadata":   metadata,
		"space_id":   spaceId,
		"created_at": time.Now().UTC().Format(time.RFC3339Nano),
	}

	point := map[string]any{
		"id":      id,
		"vector":  vector,
		"payload": payload,
	}

	req := map[string]any{
		"points": []map[string]any{point},
	}

	var rsp qdrantEnvelope[json.RawMessage]

	path := fmt.Sprintf("/collections/%s/points?wait=true", url.PathEscape(s.options.Collection))

	if err := s.do(ctx, http.MethodPut, path, req, &rsp); err != nil {
		return err
	}

	if !strings.EqualFold(rsp.Status.State, "ok") && len(rsp.Status.Error) > 0 {
		return errors.New(rsp.Status.Error)
	}

	return nil
}

func (s *qdrantStorer) Search(ctx context.Context, spaceId string, vector []float32, limit int) ([]storer.Record, error) {
	if limit < 1 {
		return nil, nil
	}

	req := map[string]any{
		"vector":       vector,
		"limit":        limit,
		"with_vector":  true,
		"with_payload": true,
		"filter": map[string]any{
			"must": []map[string]any{
				{
					"key":   "space_id",
					"match": map[string]any{"value": spaceId},
				},
			},
		},
	}

	var rsp qdrantEnvelope[[]qdrantPointResult]

	path := fmt.Sprintf("/collections/%s/points/search", url.PathEscape(s.options.Collection))

	if err := s.do(ctx, http.MethodPost, path, req, &rsp); err != nil {
		return nil, err
	}

	results := make([]storer.Record, 0, len(rsp.Result))

	for _, point := range rsp.Result {
		payload := point.Payload

		createdAt, _ := time.Parse(time.RFC3339Nano, getsafe.String(payload, "created_at"))

		rec := storer.Record{
			Id:        point.Id,
			SessionId: getsafe.String(payload, "session_id"),
			Content:   getsafe.String(payload, "content"),
			Metadata:  getsafe.Metadata(payload, "metadata"),
			Embedding: point.Vector,
			Score:     float32(point.Score),
			Space:     getsafe.String(payload, "space_id"),
			CreatedAt: createdAt,
		}

		results = append(results, rec)
	}

	return results, nil
}

func (s *qdrantStorer) do(ctx context.Context, method string, path string, req any, rsp any) error {
	u := s.options.Location + path
	var buf io.Reader
	if req != nil {
		data, err := json.Marshal(req)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(data)
	}

	request, err := http.NewRequestWithContext(ctx, method, u, buf)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	if len(s.options.ApiKey) > 0 {
		request.Header.Set("api-key", s.options.ApiKey)
		request.Header.Set("Authorization", "Bearer "+s.options.ApiKey)
	}

	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("qdrant http %d: %s", response.StatusCode, string(payload))
	}

	if rsp != nil && len(payload) > 0 {
		if err := json.Unmarshal(payload, rsp); err != nil {
			return err
		}
	}

	return nil
}

func (s *qdrantStorer) configure() error {
	exists, err := s.collectionExists()
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	return s.createCollection()
}

func (s *qdrantStorer) collectionExists() (bool, error) {
	path := fmt.Sprintf("/collections/%s", url.PathEscape(s.options.Collection))

	var rsp qdrantEnvelope[json.RawMessage]

	err := s.do(context.Background(), http.MethodGet, path, nil, &rsp)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}

	return strings.EqualFold(rsp.Status.State, "ok"), nil
}

func (s *qdrantStorer) createCollection() error {
	distance := s.options.Distance
	if len(distance) == 0 {
		distance = "Cosine"
	}
	req := map[string]any{
		"vectors": map[string]any{
			"size":     s.options.VectorSize,
			"distance": distance,
		},
	}

	path := fmt.Sprintf("/collections/%s", url.PathEscape(s.options.Collection))

	var rsp qdrantEnvelope[json.RawMessage]

	if err := s.do(context.Background(), http.MethodPut, path, req, &rsp); err != nil {
		return err
	}

	if !strings.EqualFold(rsp.Status.State, "ok") {
		return errors.New(rsp.Status.Error)
	}

	return nil
}

func NewStorer(opts ...storer.Option) storer.Storer {
	options := storer.NewOptions(opts...)

	if len(options.Location) == 0 ||
		len(options.Collection) == 0 ||
		options.VectorSize == 0 {
		panic("missing location, collection, or vector size for qdrant storer")
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	s := &qdrantStorer{
		options: options,
		client:  client,
	}

	if err := s.configure(); err != nil {
		panic(err)
	}

	return s
}
