package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/brandon-good/starql.git/llm"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type AppConfig struct {
	TestSize     int // n questions to ask the model
	LlmApi       string
	Llm          string // model to use
	QuestionFile string // file with questions
}

func (cfg *AppConfig) setupLogger() {
	logFileName := "/logs/logs_" + fmt.Sprintf("%s_%d", cfg.Llm, time.Now().Unix()) + ".log"

	logFile, err := os.Create(logFileName)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to create log file")
	}

	log.Logger = log.Output(zerolog.MultiLevelWriter(logFile, os.Stdout))
}

func main() {

	var requestsGroup sync.WaitGroup
	var resultsGroup sync.WaitGroup
	cfg, err := NewAppConfig()
	if err != nil {
		log.Panic().Err(err).Send()
	}
	cfg.setupLogger()

	log.Info().Any("cfg", cfg).Send()

	file, err := os.Open(cfg.QuestionFile)
	if err != nil {
		log.Panic().Err(err).Send()
	}
	defer file.Close()

	numWorkers := viper.GetInt("STARQL_NUMWORKERS")
	reqs := make(chan llm.Request, numWorkers*2)
	resps := make(chan llm.Response, numWorkers*2)
	ctx := context.Background()
	llmHdlr := llm.NewOllamaRequestsHandler(ctx, cfg.Llm, cfg.LlmApi, reqs, resps, numWorkers, &requestsGroup)

	llm.RunDbDaemon(ctx, resps, 5332, "localhost", viper.GetString("POSTGRES_USER"), viper.GetString("POSTGRES_PASSWORD"), llmHdlr, numWorkers, &resultsGroup, viper.GetBool("CORRECT_MODEL"))

	scanner := bufio.NewScanner(file)
	for range cfg.TestSize {
		scanner.Scan()
		line := scanner.Text()
		var record llm.BirdQuestion
		err = json.Unmarshal([]byte(line), &record)
		log.Info().Any("record", record).Send()
		llmHdlr.Request(record)
	}

	resultsGroup.Wait()
	requestsGroup.Wait()
	close(reqs)
	close(resps)
}

func NewAppConfig() (*AppConfig, error) {
	viper.AutomaticEnv()
	return &AppConfig{
		TestSize: viper.GetInt("TEST_SIZE"), LlmApi: viper.GetString("LLM_API"), Llm: viper.GetString("LLM"), QuestionFile: viper.GetString("QUESTION_FILE"),
	}, nil
}
