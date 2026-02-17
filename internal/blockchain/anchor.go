package blockchain

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/sha3"
)

type Anchor struct {
	rpcURL      string
	contract    string
	from        string
	httpClient  *http.Client
	pollEvery   time.Duration
	maxWaitTime time.Duration
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type txReceipt struct {
	BlockNumber string `json:"blockNumber"`
}

func NewAnchor(rpcURL, contract, from string) (*Anchor, error) {
	if rpcURL == "" || contract == "" || from == "" {
		return nil, errors.New("blockchain config is incomplete")
	}
	return &Anchor{
		rpcURL:      rpcURL,
		contract:    contract,
		from:        from,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		pollEvery:   400 * time.Millisecond,
		maxWaitTime: 10 * time.Second,
	}, nil
}

func (a *Anchor) Anchor(ctx context.Context, dataHashHex string) (string, uint64, error) {
	data, err := encodeAnchorCalldata(dataHashHex)
	if err != nil {
		return "", 0, err
	}

	params := []any{map[string]string{
		"from": a.from,
		"to":   a.contract,
		"data": data,
	}}

	var txHash string
	if err := a.callRPC(ctx, "eth_sendTransaction", params, &txHash); err != nil {
		return "", 0, err
	}

	deadline := time.Now().Add(a.maxWaitTime)
	for time.Now().Before(deadline) {
		var receipt *txReceipt
		err := a.callRPC(ctx, "eth_getTransactionReceipt", []any{txHash}, &receipt)
		if err == nil && receipt != nil && receipt.BlockNumber != "" {
			block, parseErr := parseHexUint64(receipt.BlockNumber)
			if parseErr != nil {
				return "", 0, parseErr
			}
			return txHash, block, nil
		}

		select {
		case <-ctx.Done():
			return "", 0, ctx.Err()
		case <-time.After(a.pollEvery):
		}
	}

	return "", 0, fmt.Errorf("anchor tx %s was not mined within %s", txHash, a.maxWaitTime)
}

func encodeAnchorCalldata(dataHashHex string) (string, error) {
	if !strings.HasPrefix(dataHashHex, "0x") {
		return "", errors.New("data hash must start with 0x")
	}
	hashBytes, err := hex.DecodeString(strings.TrimPrefix(dataHashHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("decode hash: %w", err)
	}
	if len(hashBytes) != 32 {
		return "", errors.New("data hash must be 32 bytes")
	}

	h := sha3.NewLegacyKeccak256()
	h.Write([]byte("anchor(bytes32)"))
	selector := h.Sum(nil)[:4]

	payload := make([]byte, 0, 4+32)
	payload = append(payload, selector...)
	payload = append(payload, hashBytes...)

	return "0x" + hex.EncodeToString(payload), nil
}

func (a *Anchor) callRPC(ctx context.Context, method string, params any, out any) error {
	requestBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.rpcURL, bytes.NewReader(requestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("rpc status %d: %s", resp.StatusCode, string(body))
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return err
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if string(rpcResp.Result) == "null" {
		if out == nil {
			return nil
		}
		// Allow null result for pointer outputs.
		return nil
	}

	if out == nil {
		return nil
	}

	return json.Unmarshal(rpcResp.Result, out)
}

func parseHexUint64(hexValue string) (uint64, error) {
	clean := strings.TrimPrefix(hexValue, "0x")
	if clean == "" {
		return 0, errors.New("empty hex number")
	}
	return strconv.ParseUint(clean, 16, 64)
}
