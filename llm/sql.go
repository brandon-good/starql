package llm

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/rs/zerolog/log"
)

type BirdQuestion struct {
	Id         int    `json:"question_id"`
	DbId       string `json:"db_id"`
	Question   string `json:"question"`
	Evidence   string `json:"evidence"`
	SQL        string `json:"sql"`
	Difficulty string `json:"difficulty"`
}

type PgConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DbFile   string
}

type Results struct {
	Correct bool
	Qid     int
}

type DbHandler struct {
	Responses chan Response
	Host      string
	Port      int
	User      string
	Password  string
}

func NewPgConfig(host string, port int, user string, password string, dbfile string) *PgConfig {
	return &PgConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DbFile:   dbfile,
	}
}
func RunDbDaemon(ctx context.Context, resps chan Response, port int, host, user, password string, numWorkers int, wg *sync.WaitGroup) {
	hdlr := &DbHandler{
		Responses: resps,
		Host:      host,
		Port:      port,
		User:      user,
		Password:  password,
	}

	for range numWorkers {
		wg.Add(1)
		go hdlr.Handle(ctx, wg)
	}
}

func (h *DbHandler) Handle(ctx context.Context, wg *sync.WaitGroup) {
	for {

		select {
		case req := <-h.Responses:
			cfg := NewPgConfig(h.Host, h.Port, h.User, h.Password, "bank")
			result, err := cfg.Request(ctx, req.resp.LlmSql, req.birdQ.SQL, req.birdQ.DbId)

			if err != nil {
				log.Error().Err(err).Msg("error comparing sql results")
			}

			log.Info().Int("qid", req.birdQ.Id).Bool("result", result).Msg("sql comparison result")
		case <-time.After(30 * time.Second):
			wg.Done()
		}

	}
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
		return false, err
	}

	result, err := compareRows(goldRows, llmRows)
	return result, err
}
