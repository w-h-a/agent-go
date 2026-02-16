package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
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
	storer.SanitizeEdges(metadata)

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

	records := make([]storer.Record, 0, len(rsp.Result))

	for _, point := range rsp.Result {
		rec := s.mapToStorerRecord(point)
		records = append(records, rec)
	}

	return records, nil
}

func (s *qdrantStorer) SearchNeighborhood(ctx context.Context, seedIds []string, hops int, limit int) ([]storer.Record, error) {
	if limit < 1 || len(seedIds) == 0 {
		return nil, nil
	}

	visited := map[string]struct{}{}
	var records []storer.Record

	for range hops {
		if len(seedIds) == 0 {
			break
		}

		fetchIds := make([]string, 0, len(seedIds))
		for _, id := range seedIds {
			if _, seen := visited[id]; !seen {
				fetchIds = append(fetchIds, id)
				visited[id] = struct{}{}
			}
		}

		if len(fetchIds) == 0 {
			break
		}

		points, err := s.retrievePoints(ctx, fetchIds)
		if err != nil {
			return nil, err
		}

		next := []string{}
		for _, p := range points {
			rec := s.mapToStorerRecord(p)
			records = append(records, rec)
			if len(records) >= limit {
				return records, nil
			}

			if rec.Metadata != nil {
				metadataCopy := make(map[string]any, len(rec.Metadata))
				maps.Copy(metadataCopy, rec.Metadata)
				edges := storer.SanitizeEdges(metadataCopy)
				ids := make([]string, 0, len(edges))
				for _, edge := range edges {
					ids = append(ids, edge["target"])
				}
				next = append(next, ids...)
			}
		}

		seedIds = next
	}

	return records, nil
}

func (s *qdrantStorer) retrievePoints(ctx context.Context, ids []string) ([]qdrantPointResult, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	req := map[string]any{
		"ids":          ids,
		"with_vector":  true,
		"with_payload": true,
	}

	var rsp qdrantEnvelope[[]qdrantPointResult]

	path := fmt.Sprintf("/collections/%s/points", url.PathEscape(s.options.Collection))

	if err := s.do(ctx, http.MethodPost, path, req, &rsp); err != nil {
		return nil, err
	}

	return rsp.Result, nil
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

func (s *qdrantStorer) mapToStorerRecord(point qdrantPointResult) storer.Record {
	payload := point.Payload

	createdAt, _ := time.Parse(time.RFC3339Nano, getsafe.String(payload, "created_at"))

	rec := storer.Record{
		Id:        point.Id,
		SessionId: getsafe.String(payload, "session_id"),
		Content:   getsafe.String(payload, "content"),
		Metadata:  getsafe.Metadata(payload, "metadata"),
		Embedding: point.Vector,
		Score:     float32(point.Score),
		SpaceId:   getsafe.String(payload, "space_id"),
		CreatedAt: createdAt,
	}

	return rec
}

func (s *qdrantStorer) configure(ctx context.Context) error {
	exists, err := s.collectionExists(ctx)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	return s.createCollection(ctx)
}

func (s *qdrantStorer) collectionExists(ctx context.Context) (bool, error) {
	path := fmt.Sprintf("/collections/%s", url.PathEscape(s.options.Collection))

	var rsp qdrantEnvelope[json.RawMessage]

	err := s.do(ctx, http.MethodGet, path, nil, &rsp)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}

	return strings.EqualFold(rsp.Status.State, "ok"), nil
}

func (s *qdrantStorer) createCollection(ctx context.Context) error {
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

	if err := s.do(ctx, http.MethodPut, path, req, &rsp); err != nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.configure(ctx); err != nil {
		panic(err)
	}

	return s
}
