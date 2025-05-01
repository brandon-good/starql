package llm

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/rs/zerolog/log"
)

type BirdQuestion struct {
	Id         int    `json:"question_id,omitempty"`
	DbId       string `json:"db_id"`
	Question   string `json:"question"`
	Evidence   string `json:"evidence"`
	SQL        string `json:"sql"`
	Difficulty string `json:"difficulty,omitempty"`
}

type PgConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

type Results struct {
	Correct bool
	Qid     int
}

type DbHandler struct {
	Responses       chan Response
	Host            string
	Port            int
	User            string
	Password        string
	resultsFile     string
	fineTuningFile  string
	CorrectTheModel bool
}

func NewPgConfig(host string, port int, user string, password string) *PgConfig {
	return &PgConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}
}
func RunDbDaemon(ctx context.Context, resps chan Response, port int, host, user, password string, llm *OllamaRequestsHandler, numWorkers int, wg *sync.WaitGroup, correctModel bool) {
	t := time.Now().Unix()
	hdlr := &DbHandler{
		Responses:       resps,
		Host:            host,
		Port:            port,
		User:            user,
		Password:        password,
		resultsFile:     fmt.Sprintf("/results/%d_correct_birdq.jsonl", t),
		fineTuningFile:  fmt.Sprintf("/results/%d_finetuning.jsonl", t),
		CorrectTheModel: correctModel,
	}

	for range numWorkers {
		wg.Add(1)
		go hdlr.Handle(ctx, llm, wg)
	}
}

type goldQueryError struct{}

func (h *DbHandler) Handle(ctx context.Context, llm *OllamaRequestsHandler, wg *sync.WaitGroup) {
	defer wg.Done()
	for {

		select {
		case resp := <-h.Responses:
			cfg := NewPgConfig(h.Host, h.Port, h.User, h.Password)
			result, err := cfg.Request(ctx, resp.Resp.LlmSql, resp.BirdQ.SQL, resp.BirdQ.DbId)

			if err != nil {
				log.Error().Err(err).Msg("error comparing sql results")
				var myErr *goldQueryError
				if errors.As(err, &myErr) {
					return
				}
			}
			log.Info().Int("qid", resp.BirdQ.Id).Bool("result", result).Msg("sql comparison result")
			if result {
				save(h, h.resultsFile, resp.BirdQ)

				resp.Messages[0] = OllamaMessage{
					Role:    "system",
					Content: SysPrompt,
				}
				content := fmt.Sprintf("Given the following Database Schema, convert the Question into SQL and provide your Rationale for why that SQL is correct.\n\nDatabase Schema:\n%s\n\nQuestion:\n%s", resp.Schema, resp.BirdQ.Question)

				resp.Messages[1] = OllamaMessage{
					Role:    "user",
					Content: content,
				}

				save(h, h.fineTuningFile, resp.Messages)
			} else if h.CorrectTheModel {
				llm.RequestRationaleForSql(resp.BirdQ)
			}
		}

	}
}

func save(h *DbHandler, filename string, req any) {
	f, err := os.OpenFile(filename,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		log.Panic().Err(err).Msg("error opening file")
	}
	defer f.Close()
	bq, err := json.Marshal(req)
	if err != nil {
		log.Panic().Err(err).Msg("error marshalling question")
	}

	if _, err := f.Write(append(bq, '\n')); err != nil {
		log.Panic().Err(err).Msg("error writing to answers file")
	}
}

func (m *goldQueryError) Error() string {
	return "the gold query is bad :("
}

func (p *PgConfig) Request(ctx context.Context, llmQuery, goldQuery, dbId string) (bool, error) {
	log.Debug().Str("llmQuery", llmQuery).Str("goldQuery", goldQuery).Msg("llm and gold queries")
	if llmQuery == "" {
		return false, nil
	}

	conn, err := sql.Open("sqlite3", fmt.Sprintf("/data/%s.sqlite", dbId))
	if err != nil {
		log.Panic().Err(err).Str("dbid", dbId).Msg("Failed to open database")
	}
	defer conn.Close()

	llmRows, err := conn.Query(llmQuery)
	defer func() {
		if llmRows != nil {
			llmRows.Close()
		}
	}()
	if err != nil {
		log.Error().Err(err).Msg("llmQuery failed to query the database")
		return false, err
	} else {
		log.Debug().Msg("llmQuery succeeded")
	}

	goldRows, err := conn.Query(goldQuery)
	defer func() {
		if goldRows != nil {
			goldRows.Close()
		}
	}()
	if err != nil {
		log.Error().Err(err).Msg("goldQuery failed to query the database")
		return false, &goldQueryError{}
	}

	result, err := compareRows(goldRows, llmRows)
	return result, err
}
