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

	memorymanager "github.com/w-h-a/agent/memory_manager"
)

type gomentoMemoryManager struct {
	options memorymanager.Options
	client  *http.Client
}

func (r *gomentoMemoryManager) CreateSpace(ctx context.Context, name string) (string, error) {
	bs := []byte(fmt.Sprintf(`{"name": "%s"}`, name))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/spaces", r.options.Location),
		bytes.NewReader(bs),
	)
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")

	rsp, err := r.client.Do(req)
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

func (r *gomentoMemoryManager) CreateSession(ctx context.Context, opts ...memorymanager.CreateSessionOption) (string, error) {
	options := memorymanager.NewSessionOptions(opts...)

	bs := []byte(fmt.Sprintf(`{"space_id": "%s"}`, options.SpaceId))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/sessions", r.options.Location),
		bytes.NewReader(bs),
	)
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")

	rsp, err := r.client.Do(req)
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

func (r *gomentoMemoryManager) AddShortTerm(ctx context.Context, sessionId string, role string, parts []memorymanager.Part, opts ...memorymanager.AddToShortTermOption) error {
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
		fmt.Sprintf("%s/api/v1/sessions/%s/messages", r.options.Location, sessionId),
		body,
	)
	if err != nil {
		return err
	}

	msgReq.Header.Add("Content-Type", writer.FormDataContentType())

	msgRsp, err := r.client.Do(msgReq)
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
		fmt.Sprintf("%s/api/v1/sessions/%s/extract", r.options.Location, sessionId),
		nil,
	)
	if err != nil {
		// TODO: trace/log but don't fail the whole operation here
		return nil
	}

	extractRsp, err := r.client.Do(extractReq)
	if err != nil {
		// TODO: trace/log but don't fail the whole operation here
		return nil
	}
	defer extractRsp.Body.Close()

	return nil
}

func (r *gomentoMemoryManager) FlushToLongTerm(ctx context.Context, sessionId string) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/sessions/%s/distill", r.options.Location, sessionId),
		nil,
	)
	if err != nil {
		return err
	}

	rsp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode >= 400 {
		return fmt.Errorf("distill status: %s", rsp.Status)
	}

	return nil
}

func (r *gomentoMemoryManager) ListShortTerm(ctx context.Context, sessionId string, opts ...memorymanager.ListShortTermOption) ([]memorymanager.Message, []memorymanager.Task, error) {
	options := memorymanager.NewListShortTermOptions(opts...)

	msgReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/sessions/%s/messages?limit=%d", r.options.Location, sessionId, options.Limit),
		nil,
	)
	if err != nil {
		return nil, nil, err
	}

	msgRsp, err := r.client.Do(msgReq)
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
		fmt.Sprintf("%s/api/v1/sessions/%s/tasks", r.options.Location, sessionId),
		nil,
	)
	if err != nil {
		return nil, nil, err
	}

	taskRsp, err := r.client.Do(taskReq)
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

func (r *gomentoMemoryManager) SearchLongTerm(ctx context.Context, query string, opts ...memorymanager.SearchLongTermOption) ([]memorymanager.Message, []memorymanager.Skill, error) {
	options := memorymanager.NewSearchOptions(opts...)

	params := url.Values{}
	params.Add("q", query)

	msgReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/spaces/%s/messages?%s", r.options.Location, options.SpaceId, params.Encode()),
		nil,
	)
	if err != nil {
		return nil, nil, err
	}

	msgRsp, err := r.client.Do(msgReq)
	if err != nil {
		return nil, nil, err
	}
	defer msgRsp.Body.Close()

	if msgRsp.StatusCode >= 400 {
		return nil, nil, fmt.Errorf("status: %s", msgRsp.Status)
	}

	var msgRes []memorymanager.Message

	if err := json.NewDecoder(msgRsp.Body).Decode(&msgRes); err != nil {
		return nil, nil, err
	}

	skillReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/spaces/%s/skills?%s", r.options.Location, options.SpaceId, params.Encode()),
		nil,
	)
	if err != nil {
		return nil, nil, err
	}

	skillRsp, err := r.client.Do(skillReq)
	if err != nil {
		return nil, nil, err
	}
	defer skillRsp.Body.Close()

	if skillRsp.StatusCode >= 400 {
		return nil, nil, fmt.Errorf("status: %s", skillRsp.Status)
	}

	var skillRes []memorymanager.Skill

	if err := json.NewDecoder(skillRsp.Body).Decode(&skillRes); err != nil {
		return nil, nil, err
	}

	return msgRes, skillRes, nil
}

func NewMemoryManager(opts ...memorymanager.Option) memorymanager.MemoryManager {
	options := memorymanager.NewOptions(opts...)

	r := &gomentoMemoryManager{
		options: options,
	}

	client := http.DefaultClient

	r.client = client

	return r
}
