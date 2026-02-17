package blockchain

import "testing"

func TestEncodeAnchorCalldata(t *testing.T) {
	hash := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	calldata, err := encodeAnchorCalldata(hash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calldata) != 2+(4+32)*2 {
		t.Fatalf("unexpected calldata length: %d", len(calldata))
	}
	if calldata[:10] != "0xeecdf927" {
		t.Fatalf("unexpected selector prefix: %s", calldata[:10])
	}
}

func TestEncodeAnchorCalldataValidation(t *testing.T) {
	_, err := encodeAnchorCalldata("0x1234")
	if err == nil {
		t.Fatal("expected error for short hash")
	}
}
