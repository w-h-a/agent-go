package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	_ "github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"github.com/w-h-a/agent/memory_manager/providers/storer"
	"go.nhat.io/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

var DRIVER string

func init() {
	driver, err := otelsql.Register(
		"postgres",
		otelsql.TraceQueryWithoutArgs(),
		otelsql.TraceRowsClose(),
		otelsql.TraceRowsAffected(),
		otelsql.WithSystem(semconv.DBSystemPostgreSQL),
	)
	if err != nil {
		detail := "failed to register v1 pg storer with otel"
		slog.ErrorContext(context.Background(), detail, "error", err)
		panic(detail)
	}

	DRIVER = driver
}

type postgresStorer struct {
	options storer.Options
	conn    *sql.DB
}

func (p *postgresStorer) Store(ctx context.Context, spaceId, sessionId string, content string, metadata map[string]any, vector []float32) error {
	edges := storer.SanitizeEdges(metadata)

	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO messages (
			session_id, 
			content, 
			metadata, 
			embedding,
			space_id
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id int64
	if err = p.conn.QueryRowContext(
		ctx,
		query,
		sessionId,
		content,
		metaJSON,
		pgvector.NewVector(vector),
		spaceId,
	).Scan(&id); err != nil {
		return err
	}

	idstr := strconv.FormatInt(id, 10)

	if len(edges) == 0 {
		return nil
	}
	if err := p.addEdges(ctx, idstr, edges); err != nil {
		return err
	}

	return nil
}

func (p *postgresStorer) addEdges(ctx context.Context, id string, edges []map[string]string) error {
	tx, err := p.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO message_edges (source_id, target_id, type) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, edge := range edges {
		if _, err := stmt.ExecContext(ctx, id, edge["target"], edge["type"]); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *postgresStorer) Search(ctx context.Context, spaceId string, vector []float32, limit int) ([]storer.Record, error) {
	if limit < 1 {
		return nil, nil
	}

	query := `
		SELECT 
			id, 
			session_id, 
			content, 
			metadata,
			embedding, 
			1 - (embedding <=> $2) as score,
			space_id,
			created_at, 
			updated_at
		FROM messages
		WHERE space_id = $1
		ORDER BY embedding <=> $2
		LIMIT $3
	`

	rows, err := p.conn.QueryContext(ctx, query, spaceId, pgvector.NewVector(vector), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []storer.Record

	for rows.Next() {
		var id int64
		var rec storer.Record
		var metaBytes []byte

		err := rows.Scan(
			&id,
			&rec.SessionId,
			&rec.Content,
			&metaBytes,
			&rec.Embedding,
			&rec.Score,
			&rec.SpaceId,
			&rec.CreatedAt,
			&rec.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		rec.Id = strconv.FormatInt(id, 10)

		if err := json.Unmarshal(metaBytes, &rec.Metadata); err != nil {
			rec.Metadata = make(map[string]any)
		}

		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (p *postgresStorer) SearchNeighborhood(ctx context.Context, seedIds []string, hops int, limit int) ([]storer.Record, error) {
	if limit < 1 {
		return nil, nil
	}

	query := `
    WITH RECURSIVE graph_walk AS (
        SELECT id, session_id, content, metadata, embedding, space_id, created_at, updated_at, 0 as depth
        FROM messages
        WHERE id = ANY($1::bigint[])
        
        UNION
        
        SELECT m.id, m.session_id, m.content, m.metadata, m.embedding, m.space_id, m.created_at, m.updated_at, gw.depth + 1
        FROM messages m
        INNER JOIN message_edges e ON e.target_id = m.id
        INNER JOIN graph_walk gw ON gw.id = e.source_id
        WHERE gw.depth < $2
    )
    SELECT DISTINCT ON (id) id, session_id, content, metadata, embedding, 0 as score, space_id, created_at, updated_at 
    FROM graph_walk
    LIMIT $3;
    `

	rows, err := p.conn.QueryContext(ctx, query, seedIds, hops, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []storer.Record
	for rows.Next() {
		var id int64
		var rec storer.Record
		var metaBytes []byte

		err := rows.Scan(
			&id,
			&rec.SessionId,
			&rec.Content,
			&metaBytes,
			&rec.Embedding,
			&rec.Score,
			&rec.SpaceId,
			&rec.CreatedAt,
			&rec.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		rec.Id = strconv.FormatInt(id, 10)

		if err := json.Unmarshal(metaBytes, &rec.Metadata); err != nil {
			rec.Metadata = make(map[string]any)
		}

		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil

}

func NewStorer(opts ...storer.Option) storer.Storer {
	options := storer.NewOptions(opts...)

	p := &postgresStorer{
		options: options,
	}

	// postgres://user:password@host:port/db?sslmode=disable
	conn, err := sql.Open(DRIVER, p.options.Location)
	if err != nil {
		detail := "failed to connect with postgres storer"
		slog.ErrorContext(context.Background(), detail, "error", err)
		panic(detail)
	}

	if err := conn.Ping(); err != nil {
		detail := "failed to ping with postgres storer"
		slog.ErrorContext(context.Background(), detail, "error", err)
		panic(detail)
	}

	if err := otelsql.RecordStats(conn); err != nil {
		detail := "failed to initialize postgres instrumentation for postgres storer"
		slog.ErrorContext(context.Background(), detail, "error", err)
		panic(detail)
	}

	p.conn = conn

	return p
}
