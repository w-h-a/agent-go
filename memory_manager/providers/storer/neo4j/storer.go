package neo4j

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/w-h-a/agent/memory_manager/providers/storer"
	getsafe "github.com/w-h-a/agent/util/get_safe"
)

type neo4jStorer struct {
	options storer.Options
	driver  neo4j.DriverWithContext
}

func (s *neo4jStorer) Store(ctx context.Context, spaceId string, sessionId string, content string, metadata map[string]any, vector []float32) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.options.Collection,
	})
	defer session.Close(ctx)

	edges := storer.SanitizeEdges(metadata)

	jsonMeta, _ := json.Marshal(metadata)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		createNode := `
			MERGE (m:Memory {id: $id})
			SET m.content = $content,
				m.space_id = $spaceId,
				m.session_id = $sessionId,
				m.metadata = $metadata,
				m.created_at = datetime(),
				m.embedding = $embedding
		`
		nodeParams := map[string]any{
			"id":        uuid.New().String(),
			"spaceId":   spaceId,
			"sessionId": sessionId,
			"content":   content,
			"metadata":  string(jsonMeta),
			"embedding": vector,
		}

		if _, err := tx.Run(ctx, createNode, nodeParams); err != nil {
			return nil, err
		}

		if len(edges) == 0 {
			return nil, nil
		}

		for _, edge := range edges {
			createEdge := fmt.Sprintf(`
					MATCH (source:Memory {id: $sourceId})
					MATCH (target:Memory {id: $targetId})
					MERGE (source)-[:%s]->(target)
				`, edge["type"])

			edgeParams := map[string]any{
				"sourceId": nodeParams["id"],
				"targetId": edge["target"],
			}

			if _, err := tx.Run(ctx, createEdge, edgeParams); err != nil {
				return nil, err
			}
		}

		return nil, nil
	})

	return err
}

func (s *neo4jStorer) Search(ctx context.Context, spaceId string, vector []float32, limit int) ([]storer.Record, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.options.Collection,
	})
	defer session.Close(ctx)

	query := `
		CALL db.index.vector.queryNodes($index, $k, $vec)
		YIELD node, score
		WHERE node.space_id = $spaceId
		RETURN node, score
		LIMIT $finalLimit
	`

	params := map[string]any{
		"index":      s.options.VectorIndex,
		"k":          limit * 2,
		"vec":        vector,
		"spaceId":    spaceId,
		"finalLimit": limit,
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}

	var records []storer.Record
	for result.Next(ctx) {
		if result.Err() != nil {
			return nil, result.Err()
		}
		record, err := s.mapToStorerRecord(result.Record())
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

func (s *neo4jStorer) SearchNeighborhood(ctx context.Context, seedIds []string, hops int, limit int) ([]storer.Record, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.options.Collection,
	})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH (start:Memory)
		WHERE start.id IN $seedIds
		MATCH (start)-[*1..%d]-(neighbor:Memory)
		WHERE NOT neighbor.id IN $seedIds
		RETURN DISTINCT neighbor as node, 0.0 as score
		LIMIT $limit
	`, hops)

	params := map[string]any{
		"seedIds": seedIds,
		"limit":   limit,
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}

	var records []storer.Record
	for result.Next(ctx) {
		if result.Err() != nil {
			return nil, result.Err()
		}
		record, err := s.mapToStorerRecord(result.Record())
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

func (s *neo4jStorer) mapToStorerRecord(r *neo4j.Record) (storer.Record, error) {
	nodeVal, _ := r.Get("node")

	node := neo4j.Node{}
	if n, ok := nodeVal.(neo4j.Node); ok {
		node = n
	}

	props := node.Props

	var meta map[string]any
	if v, ok := props["metadata"]; ok {
		if str, ok := v.(string); ok {
			json.Unmarshal([]byte(str), &meta)
		}
	}

	scoreVal, _ := r.Get("score")

	score := float32(0)
	if s, ok := scoreVal.(float64); ok {
		score = float32(s)
	}

	rec := storer.Record{
		Id:        getsafe.String(props, "id"),
		SpaceId:   getsafe.String(props, "space_id"),
		SessionId: getsafe.String(props, "session_id"),
		Content:   getsafe.String(props, "content"),
		Metadata:  meta,
		Score:     score,
		CreatedAt: getsafe.Time(props, "created_at"),
	}

	return rec, nil
}

func (s *neo4jStorer) configure(ctx context.Context) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.options.Collection,
	})
	defer session.Close(ctx)

	distance := s.options.Distance
	if len(distance) == 0 {
		distance = "cosine"
	}

	vectorQuery := fmt.Sprintf(
		"CREATE VECTOR INDEX %s IF NOT EXISTS "+
			"FOR (m:Memory) ON (m.embedding) "+
			"OPTIONS {indexConfig: {"+
			" `vector.dimensions`: %d,"+
			" `vector.similarity_function`: '%s'"+
			"}}",
		s.options.VectorIndex, s.options.VectorSize, distance,
	)

	if _, err := session.Run(ctx, vectorQuery, nil); err != nil {
		return fmt.Errorf("failed to create vector index: %w", err)
	}

	constraintQuery := `
		CREATE CONSTRAINT memory_id_unique IF NOT EXISTS
		FOR (m:Memory) REQUIRE m.id IS UNIQUE
	`
	if _, err := session.Run(ctx, constraintQuery, nil); err != nil {
		return fmt.Errorf("failed to create unique constraint: %w", err)
	}

	return nil
}

func NewStorer(opts ...storer.Option) storer.Storer {
	options := storer.NewOptions(opts...)

	s := &neo4jStorer{
		options: options,
	}

	driver, err := neo4j.NewDriverWithContext(
		s.options.Location,
		// TODO
		neo4j.NoAuth(),
	)
	if err != nil {
		panic(err)
	}

	s.driver = driver

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.configure(ctx); err != nil {
		panic(err)
	}

	return s
}
