package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL            string
	Port                   string
	BlockchainRPCURL       string
	BlockchainContractAddr string
	BlockchainFromAddr     string
}

func Load() Config {
	cfg := Config{
		DatabaseURL:            mustEnv("DATABASE_URL"),
		Port:                   envOr("PORT", "8080"),
		BlockchainRPCURL:       os.Getenv("BLOCKCHAIN_RPC_URL"),
		BlockchainContractAddr: os.Getenv("BLOCKCHAIN_CONTRACT_ADDRESS"),
		BlockchainFromAddr:     os.Getenv("BLOCKCHAIN_FROM_ADDRESS"),
	}
	if err := cfg.validate(); err != nil {
		slog.Error("invalid configuration", "details", err.Error())
		os.Exit(1)
	}
	return cfg
}

func (c *Config) validate() error {
	var errs []string

	if port, err := strconv.Atoi(c.Port); err != nil || port < 1 || port > 65535 {
		errs = append(errs, fmt.Sprintf("PORT=%q must be a number in range [1..65535]", c.Port))
	}

	bcSet := countNonEmpty(c.BlockchainRPCURL, c.BlockchainContractAddr, c.BlockchainFromAddr)
	if bcSet > 0 && bcSet < 3 {
		errs = append(errs, "blockchain config is incomplete: set all of BLOCKCHAIN_RPC_URL, BLOCKCHAIN_CONTRACT_ADDRESS, BLOCKCHAIN_FROM_ADDRESS or none")
	}

	if c.BlockchainContractAddr != "" && !isEthAddress(c.BlockchainContractAddr) {
		errs = append(errs, fmt.Sprintf("BLOCKCHAIN_CONTRACT_ADDRESS=%q must be 0x followed by 40 hex characters", c.BlockchainContractAddr))
	}

	if c.BlockchainFromAddr != "" && !isEthAddress(c.BlockchainFromAddr) {
		errs = append(errs, fmt.Sprintf("BLOCKCHAIN_FROM_ADDRESS=%q must be 0x followed by 40 hex characters", c.BlockchainFromAddr))
	}

	if len(errs) > 0 {
		return fmt.Errorf("  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// isEthAddress reports whether s is a valid Ethereum address (0x + 40 hex chars).
func isEthAddress(s string) bool {
	if len(s) != 42 || !strings.HasPrefix(s, "0x") {
		return false
	}
	for _, c := range s[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func countNonEmpty(vals ...string) int {
	n := 0
	for _, v := range vals {
		if v != "" {
			n++
		}
	}
	return n
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		slog.Error("missing required env", "key", key)
		os.Exit(1)
	}
	return value
}

func envOr(key, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}
