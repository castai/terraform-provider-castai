package e2e

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"

	"github.com/castai/terraform-provider-castai/v7/e2e/config"
)

var (
	cfg *config.Config
)

func TestMain(m *testing.M) {
	// Load config.
	var err error
	err = godotenv.Load()
	if err != nil {
		log.Println("unable to load dotfile", err)
	}
	cfg, err = config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	// Run all tests.
	code := m.Run()

	os.Exit(code)
}
