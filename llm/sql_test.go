package llm_test

import (
	"context"
	"os"

	"github.com/brandon-good/starql.git/llm"
	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
)

var _ = Describe("Sending queries to postgres", func() {

	It("I ask a question i get an answer", func() {
		godotenv.Load("../.env")

		conf := llm.NewPgConfig("localhost", 5332, os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), "bank")
		log.Info().Any("conf", conf).Msg("conf")
		ctx := context.Background()
		query := "select count(*) from information_schema.tables"
		query2 := "select count(*) from information_schema.tables"

		comp, err := conf.Request(ctx, query, query2)
		Expect(err).To(BeNil())
		Expect(comp).To(BeTrue())
	})

})
