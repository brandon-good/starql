package llm_test

import (
	"context"

	"github.com/brandon-good/starql.git/llm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sending queries to ollama", func() {

	When("the model is asked a question, it responds", func() {
		hdlr := llm.NewOllamaRequestsHandler(context.Background(), "llama3.2", "http://localhost:7869/api/chat", 1)
		hdlr.Request("Come up with an example SQL prompt and explain the rationale behind. Return as JSON.")
		Eventually(hdlr.Responses).Should(Receive())
	})
	When("the model is asked a question, it responds", func() {
		hdlr := llm.NewOllamaRequestsHandler(context.Background(), "llama3.2", "http://localhost:7869/api/chat", 20)
		for _ = range 30 {
			hdlr.Request("Come up with an example SQL prompt and explain the rationale behind. Return as JSON.")
		}
		Eventually(hdlr.Responses).Should(BeClosed()) // need a better condition here
	})
})
