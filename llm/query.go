package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

type OllamaRequest struct {
	Model    string                 `json:"model"`
	Messages []OllamaMessage        `json:"messages"`
	Format   map[string]interface{} `json:"format,omitempty"`
	Stream   bool                   `json:"stream"`
}

type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"` // string or ollamaresponsecontent
}

type OllamaResponse struct {
	Model   string        `json:"model"`
	Message OllamaMessage `json:"message"`
}

type OllamaResponseContent struct {
	LlmSql    string `json:"sql"`
	Rationale string `json:"rationale"`
}

type Response struct {
	Resp      OllamaResponseContent `json:"ollama_response"`
	Messages  []OllamaMessage       `json:"messages"`
	Schema    string
	BirdQ     BirdQuestion `json:"bird_question"`
	FullQuery string       `json:"full_query"`
}

type Request struct {
	req       OllamaRequest
	birdQ     BirdQuestion
	schema    string
	fullQuery string
}

type OllamaRequestsHandler struct {
	Requests  chan Request
	Responses chan Response
	Model     string
	Api       string
}

func NewOllamaRequestsHandler(ctx context.Context, model, api_url string, reqChannel chan Request, respChannel chan Response, numWorkers int, wg *sync.WaitGroup) *OllamaRequestsHandler {

	hdlr := &OllamaRequestsHandler{
		Requests:  reqChannel,
		Responses: respChannel,
		Model:     model,
		Api:       api_url,
	}

	for range numWorkers {
		wg.Add(1)
		go requestWorker(ctx, hdlr, wg)
	}
	return hdlr
}

var format = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"sql": map[string]string{"type": "string"}, "rationale": map[string]string{"type": "string"}}, "required": []string{"sql", "rationale"}}

func (h *OllamaRequestsHandler) Request(question BirdQuestion) {

	schemas := getSchemas(question.DbId)

	// TODO open that DB, get the schema, and add it to the request
	content := fmt.Sprintf("Given the following Database Schema, convert the Question into SQL and provide your Rationale for why that SQL is correct.\n\nDatabase Schema:\n%s\n\nQuestion:\n%s", schemas, question.Question)
	req := OllamaRequest{
		Model:    h.Model,
		Messages: []OllamaMessage{{Role: "system", Content: SysPrompt}, {Role: "user", Content: content}},
		Format:   format,
	}

	log.Debug().Any("messages to llm", req.Messages).Msg("pushing request onto queue")

	request := Request{
		req:       req,
		birdQ:     question,
		fullQuery: content,
		schema:    schemas,
	}

	h.Requests <- request
}

func (h *OllamaRequestsHandler) RequestRationaleForSql(question BirdQuestion) {

	schemas := getSchemas(question.DbId)
	// TODO open that DB, get the schema, and add it to the request
	sys := "You will be given a Database Schema, Question, and the correct SQL answer. Your task is to provide a Rationale as to why the SQL answer is correct. Repeat the SQL back to me exactly as I provide it."
	content := fmt.Sprintf(`Database Schema:
	%s
	
	Question:
	%s
	
	Correct SQL Answer:
	%s`, schemas, question.Question, question.SQL)
	req := OllamaRequest{
		Model:    h.Model,
		Messages: []OllamaMessage{{Role: "system", Content: sys}, {Role: "user", Content: content}},
		Format:   format,
	}

	log.Debug().Any("messages to llm", req.Messages).Msg("Requesting rationale provided correct SQL ")

	request := Request{
		req:       req,
		birdQ:     question,
		fullQuery: content,
		schema:    schemas,
	}

	go func() { h.Requests <- request }()

}

func (h *OllamaRequestsHandler) SyncRequest(role, prompt string) (string, error) {
	req := Request{}
	req.req = OllamaRequest{
		Model:    h.Model,
		Messages: []OllamaMessage{{Role: "control", Content: "thinking"}},
	}
	client := resty.New()
	ollamaResponse := &OllamaResponse{}
	resp, err := client.R().
		SetBody(req.req).
		SetResult(ollamaResponse).
		Post(h.Api)

	if resp.IsSuccess() {
		log.Debug().Any("user req", req.req.Messages).Str("ollama resp", resp.String()).Send()
		log.Debug().Any("ollama content", ollamaResponse.Message.Content).Send()
		return ollamaResponse.Message.Content, nil

	} else {
		log.Error().Str("ollama resp", resp.String()).Send()
		if err != nil {
			log.Error().Err(err).Msg("issue posting to ollama.")
		}
		return "", err
	}

}

func requestWorker(ctx context.Context, hdlr *OllamaRequestsHandler, wg *sync.WaitGroup) {
	log.Debug().Msg("request worker created")
	defer wg.Done()
	for {
		select {
		case req := <-hdlr.Requests:
			client := resty.New()
			ollamaResponse := &OllamaResponse{}
			resp, err := client.R().
				SetBody(req.req).
				SetResult(ollamaResponse).
				Post(hdlr.Api)

			if resp.IsSuccess() {
				log.Debug().Any("ollama content", ollamaResponse.Message.Content).Send()
				content := &OllamaResponseContent{}
				json.Unmarshal([]byte(ollamaResponse.Message.Content), content)
				if req.schema == "" {
					panic("schema is empty")
				}
				response := Response{
					Resp:      *content,
					Messages:  append(req.req.Messages, ollamaResponse.Message),
					BirdQ:     req.birdQ,
					FullQuery: req.fullQuery,
					Schema:    req.schema,
				}
				hdlr.Responses <- response
			} else {
				log.Error().Str("ollama resp", resp.String()).Send()
			}
			if err != nil {
				log.Error().Err(err).Msg("issue posting to ollama.")
			}

		}
	}
}
