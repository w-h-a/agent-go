package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/w-h-a/agent/retriever"
	"go.nhat.io/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
		detail := "failed to register v1 pg persister with otel"
		slog.ErrorContext(context.Background(), detail, "error", err)
		panic(detail)
	}

	DRIVER = driver
}

type postgresRetriever struct {
	options retriever.Options
	conn    *sql.DB
	embeddings.Embedder
	shortTerm map[string][]retriever.Message
	mtx       sync.RWMutex
}

func (r *postgresRetriever) CreateSpace(ctx context.Context, name string) (string, error) {
	return "", nil
}

func (r *postgresRetriever) CreateSession(ctx context.Context, opts ...retriever.CreateSessionOption) (string, error) {
	return fmt.Sprintf("session-%d", time.Now().Unix()), nil
}

func (r *postgresRetriever) AddShortTerm(ctx context.Context, sessionId string, role string, parts []retriever.Part, opts ...retriever.AddToShortTermOption) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	var sb strings.Builder
	for _, p := range parts {
		if len(p.Text) > 0 {
			sb.WriteString(p.Text + "\n")
		}
	}

	text := sb.String()
	if len(strings.TrimSpace(text)) == 0 {
		return nil
	}

	vec, err := r.EmbedQuery(ctx, text)
	if err != nil {
		return err
	}

	record := retriever.Message{
		SessionId: sessionId,
		Role:      role,
		Parts:     parts,
		Embedding: vec,
	}

	r.shortTerm[sessionId] = append(r.shortTerm[sessionId], record)

	if len(r.shortTerm[sessionId]) > r.options.ShortTermMemorySize {
		r.shortTerm[sessionId] = r.shortTerm[sessionId][len(r.shortTerm[sessionId])-r.options.ShortTermMemorySize:]
	}

	return nil
}

func (r *postgresRetriever) FlushToLongTerm(ctx context.Context, sessionId string) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	records := r.shortTerm[sessionId]

	for _, record := range records {
		if err := r.store(ctx, record); err != nil {
			return err
		}
	}

	delete(r.shortTerm, sessionId)

	return nil
}

func (r *postgresRetriever) ListShortTerm(ctx context.Context, sessionId string, opts ...retriever.ListShortTermOption) ([]retriever.Message, []retriever.Task, error) {
	r.mtx.RLock()
	defer r.mtx.RUnlock()

	cpy := make([]retriever.Message, len(r.shortTerm[sessionId]))
	copy(cpy, r.shortTerm[sessionId])

	return cpy, nil, nil
}

func (r *postgresRetriever) SearchLongTerm(ctx context.Context, query string, opts ...retriever.SearchLongTermOption) ([]retriever.Message, []retriever.Skill, error) {
	options := retriever.NewSearchOptions(opts...)

	vec, err := r.EmbedQuery(ctx, query)
	if err != nil {
		return nil, nil, err
	}

	msgs, skills, err := r.search(ctx, vec, options.Limit)
	if err != nil {
		return nil, nil, err
	}

	return msgs, skills, nil
}

func (r *postgresRetriever) store(ctx context.Context, msg retriever.Message) error {
	partsJson, err := json.Marshal(msg.Parts)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO messages (session_id, role, parts, embedding)
		VALUES ($1, $2, $3, $4)
	`

	if _, err := r.conn.ExecContext(
		ctx,
		query,
		msg.SessionId,
		msg.Role,
		partsJson,
		pgvector.NewVector(msg.Embedding),
	); err != nil {
		return err
	}

	return nil
}

func (r *postgresRetriever) search(ctx context.Context, vec []float32, limit int) ([]retriever.Message, []retriever.Skill, error) {
	query := `
		SELECT id, session_id, role, parts
		FROM messages
		ORDER BY embedding <=> $1
		LIMIT $2
	`

	rows, err := r.conn.QueryContext(ctx, query, pgvector.NewVector(vec), limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var msgs []retriever.Message
	for rows.Next() {
		var m retriever.Message
		var partsBytes []byte
		if err := rows.Scan(&m.Id, &m.SessionId, &m.Role, &partsBytes); err != nil {
			return nil, nil, err
		}
		if err := json.Unmarshal(partsBytes, &m.Parts); err != nil {
			return nil, nil, err
		}
		msgs = append(msgs, m)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return msgs, nil, nil
}

func NewRetriever(opts ...retriever.Option) retriever.Retriever {
	options := retriever.NewOptions(opts...)

	r := &postgresRetriever{
		options:   options,
		shortTerm: map[string][]retriever.Message{},
		mtx:       sync.RWMutex{},
	}

	// postgres://user:password@host:port/db?sslmode=disable
	conn, err := sql.Open(DRIVER, r.options.Location)
	if err != nil {
		detail := "failed to connect with postgres retriever"
		slog.ErrorContext(context.Background(), detail, "error", err)
		panic(detail)
	}

	if err := conn.Ping(); err != nil {
		detail := "failed to ping with postgres retriever"
		slog.ErrorContext(context.Background(), detail, "error", err)
		panic(detail)
	}

	if err := otelsql.RecordStats(conn); err != nil {
		detail := "failed to initialize postgres instrumentation for postgres retriever"
		slog.ErrorContext(context.Background(), detail, "error", err)
		panic(detail)
	}

	r.conn = conn

	llmOpts := []openai.Option{
		openai.WithToken(options.ApiKey),
		openai.WithModel(options.Model),
		openai.WithHTTPClient(&http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}),
	}

	llm, err := openai.New(llmOpts...)
	if err != nil {
		detail := "failed to initialize model for openai embedder"
		slog.ErrorContext(context.Background(), detail, "error", err)
		panic(detail)
	}

	emb, err := embeddings.NewEmbedder(llm)
	if err != nil {
		detail := "failed to initialize embedder for openai embedder"
		slog.ErrorContext(context.Background(), detail, "error", err)
		panic(detail)
	}

	r.Embedder = emb

	return r
}
