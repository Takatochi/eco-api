package config

import "testing"

func TestIsEthAddress(t *testing.T) {
	valid := []string{
		"0x0000000000000000000000000000000000000000",
		"0xAbCdEf1234567890AbCdEf1234567890AbCdEf12",
		"0xffffffffffffffffffffffffffffffffffffffff",
		"0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
	}
	for _, s := range valid {
		if !isEthAddress(s) {
			t.Errorf("expected valid address: %q", s)
		}
	}

	invalid := []string{
		"",
		"0x",
		"0x1234",
		"1234567890123456789012345678901234567890",   // missing 0x prefix
		"0x00000000000000000000000000000000000000GG", // invalid hex chars
		"0x000000000000000000000000000000000000000000", // 43 chars, too long
		"0X0000000000000000000000000000000000000000", // uppercase 0X not accepted
	}
	for _, s := range invalid {
		if isEthAddress(s) {
			t.Errorf("expected invalid address: %q", s)
		}
	}
}

func TestValidatePort(t *testing.T) {
	cases := []struct {
		port  string
		valid bool
	}{
		{"8080", true},
		{"1", true},
		{"65535", true},
		{"0", false},
		{"65536", false},
		{"abc", false},
		{"", false},
		{"-1", false},
	}
	for _, tc := range cases {
		cfg := &Config{Port: tc.port}
		err := cfg.validate()
		if tc.valid && err != nil {
			t.Errorf("port %q: expected valid, got error: %v", tc.port, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("port %q: expected error, got nil", tc.port)
		}
	}
}

func TestValidateBlockchainPartial(t *testing.T) {
	addr := "0x0000000000000000000000000000000000000000"

	t.Run("all three set is valid", func(t *testing.T) {
		cfg := &Config{
			Port:                   "8080",
			BlockchainRPCURL:       "http://localhost:8545",
			BlockchainContractAddr: addr,
			BlockchainFromAddr:     addr,
		}
		if err := cfg.validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("none set is valid", func(t *testing.T) {
		cfg := &Config{Port: "8080"}
		if err := cfg.validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("only rpc url set is invalid", func(t *testing.T) {
		cfg := &Config{Port: "8080", BlockchainRPCURL: "http://localhost:8545"}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for partial blockchain config")
		}
	})

	t.Run("rpc url and contract set is invalid", func(t *testing.T) {
		cfg := &Config{
			Port:                   "8080",
			BlockchainRPCURL:       "http://localhost:8545",
			BlockchainContractAddr: addr,
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for partial blockchain config")
		}
	})
}

func TestValidateBlockchainAddressFormat(t *testing.T) {
	addr := "0x0000000000000000000000000000000000000000"
	badAddr := "0xBAD"

	t.Run("invalid contract address", func(t *testing.T) {
		cfg := &Config{
			Port:                   "8080",
			BlockchainRPCURL:       "http://localhost:8545",
			BlockchainContractAddr: badAddr,
			BlockchainFromAddr:     addr,
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for invalid contract address")
		}
	})

	t.Run("invalid from address", func(t *testing.T) {
		cfg := &Config{
			Port:                   "8080",
			BlockchainRPCURL:       "http://localhost:8545",
			BlockchainContractAddr: addr,
			BlockchainFromAddr:     badAddr,
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for invalid from address")
		}
	})
}
