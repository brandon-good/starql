package llm_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestLlm(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: GinkgoWriter})
	SetDefaultEventuallyTimeout(time.Second * 10)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Llm Suite")

}
