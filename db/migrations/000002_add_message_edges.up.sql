CREATE TABLE IF NOT EXISTS message_edges (
    source_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    target_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    type TEXT NOT NULL, 
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (source_id, target_id, type)
);

CREATE INDEX IF NOT EXISTS idx_messages_edges_source ON messages_edges(source_id);
CREATE INDEX IF NOT EXISTS idx_messages_edges_target ON messages_edges(target_id);