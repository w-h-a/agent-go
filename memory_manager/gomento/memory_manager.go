package gomento

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sync"

	memorymanager "github.com/w-h-a/agent/memory_manager"
)

type gomentoMemoryManager struct {
	options       memorymanager.Options
	client        *http.Client
	sessionSpaces map[string]string
	mtx           sync.RWMutex
}

func (m *gomentoMemoryManager) CreateSpace(ctx context.Context, name string) (string, error) {
	bs := []byte(fmt.Sprintf(`{"name": "%s"}`, name))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/spaces", m.options.Location),
		bytes.NewReader(bs),
	)
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")

	rsp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode >= 400 {
		return "", fmt.Errorf("status: %s", rsp.Status)
	}

	var res struct {
		Id string `json:"id"`
	}

	if err := json.NewDecoder(rsp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.Id, nil
}

func (m *gomentoMemoryManager) CreateSession(ctx context.Context, opts ...memorymanager.CreateSessionOption) (string, error) {
	options := memorymanager.NewCreateSessionOptions(opts...)

	bs := []byte(fmt.Sprintf(`{"space_id": "%s"}`, options.SpaceId))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/sessions", m.options.Location),
		bytes.NewReader(bs),
	)
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")

	rsp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode >= 400 {
		return "", fmt.Errorf("status: %s", rsp.Status)
	}

	var res struct {
		Id string `json:"id"`
	}

	if err := json.NewDecoder(rsp.Body).Decode(&res); err != nil {
		return "", err
	}

	m.mtx.Lock()
	m.sessionSpaces[res.Id] = options.SpaceId
	m.mtx.Unlock()

	return res.Id, nil
}

func (m *gomentoMemoryManager) AddShortTerm(ctx context.Context, sessionId string, role string, parts []memorymanager.Part, opts ...memorymanager.AddToShortTermOption) error {
	options := memorymanager.NewAddToShortTermOptions(opts...)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("role", role); err != nil {
		return err
	}

	partsJson, err := json.Marshal(parts)
	if err != nil {
		return err
	}

	if err := writer.WriteField("parts", string(partsJson)); err != nil {
		return err
	}

	for key, file := range options.Files {
		partWriter, err := writer.CreateFormFile(key, file.Name)
		if err != nil {
			return err
		}
		if _, err := io.Copy(partWriter, file.Reader); err != nil {
			return err
		}
	}

	if err := writer.Close(); err != nil {
		return err
	}

	msgReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/sessions/%s/messages", m.options.Location, sessionId),
		body,
	)
	if err != nil {
		return err
	}

	msgReq.Header.Add("Content-Type", writer.FormDataContentType())

	msgRsp, err := m.client.Do(msgReq)
	if err != nil {
		return err
	}
	defer msgRsp.Body.Close()

	if msgRsp.StatusCode >= 400 {
		return fmt.Errorf("status: %s", msgRsp.Status)
	}

	extractReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/sessions/%s/extract", m.options.Location, sessionId),
		nil,
	)
	if err != nil {
		// TODO: trace/log but don't fail the whole operation here
		return nil
	}

	extractRsp, err := m.client.Do(extractReq)
	if err != nil {
		// TODO: trace/log but don't fail the whole operation here
		return nil
	}
	defer extractRsp.Body.Close()

	return nil
}

func (m *gomentoMemoryManager) ListShortTerm(ctx context.Context, sessionId string, opts ...memorymanager.ListShortTermOption) ([]memorymanager.Message, []memorymanager.Task, error) {
	options := memorymanager.NewListShortTermOptions(opts...)

	msgReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/sessions/%s/messages?limit=%d", m.options.Location, sessionId, options.Limit),
		nil,
	)
	if err != nil {
		return nil, nil, err
	}

	msgRsp, err := m.client.Do(msgReq)
	if err != nil {
		return nil, nil, err
	}
	defer msgRsp.Body.Close()

	if msgRsp.StatusCode >= 400 {
		return nil, nil, fmt.Errorf("status: %s", msgRsp.Status)
	}

	var msgRes struct {
		Items []memorymanager.Message `json:"items"`
	}

	if err := json.NewDecoder(msgRsp.Body).Decode(&msgRes); err != nil {
		return nil, nil, err
	}

	taskReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/sessions/%s/tasks", m.options.Location, sessionId),
		nil,
	)
	if err != nil {
		return nil, nil, err
	}

	taskRsp, err := m.client.Do(taskReq)
	if err != nil {
		return nil, nil, err
	}
	defer taskRsp.Body.Close()

	if taskRsp.StatusCode >= 400 {
		return nil, nil, fmt.Errorf("status: %s", taskRsp.Status)
	}

	var taskRes struct {
		Items []memorymanager.Task `json:"items"`
	}

	if err := json.NewDecoder(taskRsp.Body).Decode(&taskRes); err != nil {
		return nil, nil, err
	}

	return msgRes.Items, taskRes.Items, nil
}

func (m *gomentoMemoryManager) FlushToLongTerm(ctx context.Context, sessionId string) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/sessions/%s/distill", m.options.Location, sessionId),
		nil,
	)
	if err != nil {
		return err
	}

	rsp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode >= 400 {
		return fmt.Errorf("distill status: %s", rsp.Status)
	}

	return nil
}

func (m *gomentoMemoryManager) SearchLongTerm(ctx context.Context, sessionId string, query string, opts ...memorymanager.SearchLongTermOption) ([]memorymanager.Message, []memorymanager.Skill, error) {
	// TODO: gomento should account for limit
	// options := memorymanager.NewSearchOptions(opts...)

	m.mtx.Lock()
	spaceId, exists := m.sessionSpaces[sessionId]
	if !exists {
		fetchedSpaceId, err := m.fetchSessionSpaceId(ctx, sessionId)
		if err != nil {
			m.mtx.Unlock()
			return nil, nil, fmt.Errorf("failed to resolve session space: %w", err)
		}
		spaceId = fetchedSpaceId
		m.sessionSpaces[sessionId] = spaceId
	}
	m.mtx.Unlock()

	if len(spaceId) == 0 {
		return []memorymanager.Message{}, []memorymanager.Skill{}, nil
	}

	msgs, err := m.searchSpaceForMessages(ctx, spaceId, query)
	if err != nil {
		return nil, nil, err
	}

	skills, err := m.searchSpaceForSkills(ctx, spaceId, query)
	if err != nil {
		return nil, nil, err
	}

	return msgs, skills, nil
}

func (m *gomentoMemoryManager) fetchSessionSpaceId(ctx context.Context, sessionId string) (string, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/sessions/%s", m.options.Location, sessionId),
		nil,
	)
	if err != nil {
		return "", err
	}

	rsp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode >= 400 {
		return "", fmt.Errorf("status: %s", rsp.Status)
	}

	var res struct {
		Id      string `json:"id"`
		SpaceId string `json:"space_id"`
	}

	if err := json.NewDecoder(rsp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.SpaceId, nil
}

func (m *gomentoMemoryManager) searchSpaceForMessages(ctx context.Context, spaceId string, query string) ([]memorymanager.Message, error) {
	params := url.Values{}
	params.Add("q", query)

	msgReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/spaces/%s/messages?%s", m.options.Location, spaceId, params.Encode()),
		nil,
	)
	if err != nil {
		return nil, err
	}

	msgRsp, err := m.client.Do(msgReq)
	if err != nil {
		return nil, err
	}
	defer msgRsp.Body.Close()

	if msgRsp.StatusCode >= 400 {
		return nil, fmt.Errorf("status: %s", msgRsp.Status)
	}

	var msgRes []memorymanager.Message

	if err := json.NewDecoder(msgRsp.Body).Decode(&msgRes); err != nil {
		return nil, err
	}
	return msgRes, nil
}

func (m *gomentoMemoryManager) searchSpaceForSkills(ctx context.Context, spaceId string, query string) ([]memorymanager.Skill, error) {
	params := url.Values{}
	params.Add("q", query)

	skillReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/spaces/%s/skills?%s", m.options.Location, spaceId, params.Encode()),
		nil,
	)
	if err != nil {
		return nil, err
	}

	skillRsp, err := m.client.Do(skillReq)
	if err != nil {
		return nil, err
	}
	defer skillRsp.Body.Close()

	if skillRsp.StatusCode >= 400 {
		return nil, fmt.Errorf("status: %s", skillRsp.Status)
	}

	var skillRes []memorymanager.Skill

	if err := json.NewDecoder(skillRsp.Body).Decode(&skillRes); err != nil {
		return nil, err
	}

	return skillRes, nil
}

func NewMemoryManager(opts ...memorymanager.Option) memorymanager.MemoryManager {
	options := memorymanager.NewOptions(opts...)

	r := &gomentoMemoryManager{
		options:       options,
		sessionSpaces: map[string]string{},
		mtx:           sync.RWMutex{},
	}

	client := http.DefaultClient

	r.client = client

	return r
}
