package main

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type AppConfig struct {
	DbUrl    string // database to query
	TestSize int    // n questions to ask the model
	LlmApi   string
	
}

func main() {
	cfg, err := NewAppConfig()
	if err != nil {
		log.Panic().Err(err).Send()
	}

	fmt.Printf("%+v", cfg)

}

func NewAppConfig() (*AppConfig, error) {
	viper.AutomaticEnv()
	return &AppConfig{
		TestSize: viper.GetInt("TEST_SIZE"), DbUrl: viper.GetString("DB_URL"), LlmApi: viper.GetString("LLM_API"),
	}, nil
}
