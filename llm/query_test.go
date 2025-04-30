package llm_test

import (
	"context"

	"github.com/brandon-good/starql.git/llm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sending queries to ollama", func() {

	It("the model is asked a question, it responds", func() {
		req := make(chan llm.Request, 1)
		res := make(chan llm.Response, 1)
		hdlr := llm.NewOllamaRequestsHandler(context.Background(), "llama3.2", "http://localhost:11434/api/chat", req, res, 1)
		hdlr.Request(llm.BirdQuestion{
			Id:         1,
			DbId:       "1",
			Question:   "What is the capital of France?",
			Evidence:   "Paris is the capital of France.",
			SQL:        "SELECT capital FROM countries WHERE name = 'France'",
			Difficulty: "easy",
		})
		Eventually(hdlr.Responses).Should(Receive())
	})
	It("the model is asked a bunch of questions, it responds", func() {
		req := make(chan llm.Request, 10)
		res := make(chan llm.Response, 10)
		hdlr := llm.NewOllamaRequestsHandler(context.Background(), "llama3.2", "http://localhost:11434/api/chat", req, res, 10)
		for _ = range 10 {
			hdlr.Request(llm.BirdQuestion{
				Id:         1,
				DbId:       "1",
				Question:   "What is the capital of France?",
				Evidence:   "Paris is the capital of France.",
				SQL:        "SELECT capital FROM countries WHERE name = 'France'",
				Difficulty: "easy",
			})
		}
		Expect(true).To(BeFalse())
		Eventually(hdlr.Responses).Should(HaveLen(10)) // need a better condition here
	})
})
