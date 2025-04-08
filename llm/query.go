package llm

import (
	"context"
	"encoding/json"

	"github.com/go-resty/resty/v2"
	"github.com/invopop/jsonschema"
	"github.com/rs/zerolog/log"
)

type OllamaRequest struct {
	Model    string            `json:"model"`
	Messages []OllamaMessage   `json:"messages"`
	Format   jsonschema.Schema `json:"format,omitempty"`
	Stream   bool              `json:"stream"`
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
	Sql       string `json:"sql"`
	Rationale string `json:"rationale"`
}

type OllamaRequestsHandler struct {
	Requests  chan OllamaRequest
	Responses chan OllamaResponseContent
	Model     string
	Api       string
}

func NewOllamaRequestsHandler(ctx context.Context, model, api_url string, numWorkers int) *OllamaRequestsHandler {

	hdlr := &OllamaRequestsHandler{
		Requests:  make(chan OllamaRequest),
		Responses: make(chan OllamaResponseContent),
		Model:     model,
		Api:       api_url,
	}

	for _ = range numWorkers {
		go requestWorker(ctx, hdlr.Requests, hdlr.Responses, hdlr.Api)
	}
	return hdlr
}

func (h *OllamaRequestsHandler) Request(query string) {
	log.Debug().Msg("new req!")

	req := OllamaRequest{
		Model:    h.Model,
		Messages: []OllamaMessage{{Role: "user", Content: query}},
		Format:   *jsonschema.Reflect(&OllamaResponseContent{}),
	}

	h.Requests <- req
}

func requestWorker(ctx context.Context, requestChan chan OllamaRequest, responseChan chan OllamaResponseContent, api string) {
	log.Debug().Msg("request worker created")
	for {
		select {
		case req := <-requestChan:
			log.Debug().Any("request", req).Msg("request received")

			client := resty.New()
			ollamaResponse := &OllamaResponse{}
			resp, err := client.R().
				SetBody(req).
				SetResult(ollamaResponse).
				Post(api)

			if resp.IsSuccess() {
				log.Debug().Str("ollama resp", resp.String()).Send()
				log.Debug().Any("ollama content", ollamaResponse.Message.Content).Send()
				content := &OllamaResponseContent{}
				json.Unmarshal([]byte(ollamaResponse.Message.Content), content)
				responseChan <- *content
			} else {
				log.Error().Str("ollama resp", resp.String()).Send()
			}
			if err != nil {
				log.Error().Err(err).Msg("issue posting to ollama.")
			}

		case <-ctx.Done():
			break
		}
	}
}
