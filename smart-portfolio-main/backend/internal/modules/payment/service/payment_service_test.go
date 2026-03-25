package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// ---------------------------------------------------------------------------
// VerifyWebhookSignature
// ---------------------------------------------------------------------------

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	secret := "whsec_test_secret_12345"
	svc := &paymentService{
		webhookSecret: secret,
	}

	payload := []byte(`{"event":"payment.captured","id":"evt_123","payload":{"payment":{"entity":{"id":"pay_abc","email":"test@example.com","amount":50000,"currency":"INR"}}}}`)

	// Compute the expected HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	if !svc.VerifyWebhookSignature(payload, signature) {
		t.Fatal("expected VerifyWebhookSignature to return true for valid signature")
	}
}

func TestVerifyWebhookSignature_InvalidSignature(t *testing.T) {
	secret := "whsec_test_secret_12345"
	svc := &paymentService{
		webhookSecret: secret,
	}

	payload := []byte(`{"event":"payment.captured"}`)
	wrongSignature := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

	if svc.VerifyWebhookSignature(payload, wrongSignature) {
		t.Fatal("expected VerifyWebhookSignature to return false for invalid signature")
	}
}

func TestVerifyWebhookSignature_TamperedPayload(t *testing.T) {
	secret := "whsec_my_secret"
	svc := &paymentService{
		webhookSecret: secret,
	}

	originalPayload := []byte(`{"event":"payment.captured","id":"evt_123","payload":{"payment":{"entity":{"amount":50000}}}}`)

	// Sign the original payload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(originalPayload)
	signature := hex.EncodeToString(mac.Sum(nil))

	// Tamper with the payload (change amount)
	tamperedPayload := []byte(`{"event":"payment.captured","id":"evt_123","payload":{"payment":{"entity":{"amount":99999}}}}`)

	if svc.VerifyWebhookSignature(tamperedPayload, signature) {
		t.Fatal("expected VerifyWebhookSignature to return false for tampered payload")
	}
}

func TestVerifyWebhookSignature_EmptySecret(t *testing.T) {
	svc := &paymentService{
		webhookSecret: "",
	}

	payload := []byte(`{"event":"payment.captured"}`)
	signature := "anything"

	if svc.VerifyWebhookSignature(payload, signature) {
		t.Fatal("expected VerifyWebhookSignature to return false when secret is empty")
	}
}

func TestVerifyWebhookSignature_EmptyPayload(t *testing.T) {
	svc := &paymentService{
		webhookSecret: "my_secret",
	}

	if svc.VerifyWebhookSignature([]byte{}, "some_sig") {
		t.Fatal("expected VerifyWebhookSignature to return false for empty payload")
	}
}

func TestVerifyWebhookSignature_EmptySignature(t *testing.T) {
	svc := &paymentService{
		webhookSecret: "my_secret",
	}

	payload := []byte(`{"event":"payment.captured"}`)

	if svc.VerifyWebhookSignature(payload, "") {
		t.Fatal("expected VerifyWebhookSignature to return false for empty signature")
	}
}

func TestVerifyWebhookSignature_NilPayload(t *testing.T) {
	svc := &paymentService{
		webhookSecret: "my_secret",
	}

	if svc.VerifyWebhookSignature(nil, "some_sig") {
		t.Fatal("expected VerifyWebhookSignature to return false for nil payload")
	}
}

func TestVerifyWebhookSignature_CaseSensitiveSignature(t *testing.T) {
	secret := "test_secret"
	svc := &paymentService{
		webhookSecret: secret,
	}

	payload := []byte(`{"test":"data"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	correctSig := hex.EncodeToString(mac.Sum(nil))

	// hex.EncodeToString returns lowercase; try uppercase
	upperSig := ""
	for _, c := range correctSig {
		if c >= 'a' && c <= 'f' {
			upperSig += string(c - 32) // to uppercase
		} else {
			upperSig += string(c)
		}
	}

	// HMAC comparison is byte-level, so uppercase should fail
	if svc.VerifyWebhookSignature(payload, upperSig) {
		t.Fatal("expected VerifyWebhookSignature to return false for uppercase hex signature")
	}
}

func TestVerifyWebhookSignature_DifferentSecrets(t *testing.T) {
	payload := []byte(`{"event":"payment.captured","id":"evt_456"}`)

	// Sign with secret A
	secretA := "secret_a"
	macA := hmac.New(sha256.New, []byte(secretA))
	macA.Write(payload)
	sigA := hex.EncodeToString(macA.Sum(nil))

	// Verify with secret B — should fail
	svc := &paymentService{
		webhookSecret: "secret_b",
	}

	if svc.VerifyWebhookSignature(payload, sigA) {
		t.Fatal("expected VerifyWebhookSignature to return false when secrets differ")
	}
}

func TestVerifyWebhookSignature_LargePayload(t *testing.T) {
	secret := "large_payload_secret"
	svc := &paymentService{
		webhookSecret: secret,
	}

	// Simulate a large webhook payload (~100KB)
	largeData := make([]byte, 100*1024)
	for i := range largeData {
		largeData[i] = byte('A' + (i % 26))
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(largeData)
	signature := hex.EncodeToString(mac.Sum(nil))

	if !svc.VerifyWebhookSignature(largeData, signature) {
		t.Fatal("expected VerifyWebhookSignature to return true for valid large payload")
	}
}

// ---------------------------------------------------------------------------
// IsDuplicateEventError
// ---------------------------------------------------------------------------

func TestIsDuplicateEventError_True(t *testing.T) {
	err := &DuplicateEventError{EventID: "evt_abc123"}
	if !IsDuplicateEventError(err) {
		t.Fatal("expected IsDuplicateEventError to return true")
	}
}

func TestIsDuplicateEventError_False(t *testing.T) {
	err := &DuplicateEventError{EventID: "evt_abc123"}
	_ = err // suppress unused
	if IsDuplicateEventError(nil) {
		t.Fatal("expected IsDuplicateEventError to return false for nil")
	}
}

func TestIsDuplicateEventError_OtherError(t *testing.T) {
	err := &paymentError{msg: "some other error"}
	if IsDuplicateEventError(err) {
		t.Fatal("expected IsDuplicateEventError to return false for non-DuplicateEventError")
	}
}

func TestDuplicateEventError_Message(t *testing.T) {
	err := &DuplicateEventError{EventID: "evt_xyz789"}
	expected := "duplicate webhook event: evt_xyz789"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}

// ---------------------------------------------------------------------------
// isDuplicateKeyError
// ---------------------------------------------------------------------------

func TestIsDuplicateKeyError_SQLState23505(t *testing.T) {
	err := &paymentError{msg: "ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)"}
	if !isDuplicateKeyError(err) {
		t.Fatal("expected isDuplicateKeyError to return true for SQLSTATE 23505")
	}
}

func TestIsDuplicateKeyError_DuplicateKeyMessage(t *testing.T) {
	err := &paymentError{msg: "duplicate key value violates unique constraint \"outbox_events_event_id_key\""}
	if !isDuplicateKeyError(err) {
		t.Fatal("expected isDuplicateKeyError to return true for 'duplicate key' message")
	}
}

func TestIsDuplicateKeyError_UniqueConstraintMessage(t *testing.T) {
	err := &paymentError{msg: "unique constraint violation on event_id"}
	if !isDuplicateKeyError(err) {
		t.Fatal("expected isDuplicateKeyError to return true for 'unique constraint' message")
	}
}

func TestIsDuplicateKeyError_UnrelatedError(t *testing.T) {
	err := &paymentError{msg: "connection refused"}
	if isDuplicateKeyError(err) {
		t.Fatal("expected isDuplicateKeyError to return false for unrelated error")
	}
}

func TestIsDuplicateKeyError_Nil(t *testing.T) {
	if isDuplicateKeyError(nil) {
		t.Fatal("expected isDuplicateKeyError to return false for nil")
	}
}

func TestIsDuplicateKeyError_EmptyMessage(t *testing.T) {
	err := &paymentError{msg: ""}
	if isDuplicateKeyError(err) {
		t.Fatal("expected isDuplicateKeyError to return false for empty message")
	}
}

// ---------------------------------------------------------------------------
// truncate helper
// ---------------------------------------------------------------------------

func TestTruncate_ShortString(t *testing.T) {
	result := truncate("hello", 10)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestTruncate_ExactLength(t *testing.T) {
	result := truncate("hello", 5)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestTruncate_LongString(t *testing.T) {
	result := truncate("hello world", 5)
	if result != "hello..." {
		t.Errorf("expected 'hello...', got %q", result)
	}
}

func TestTruncate_EmptyString(t *testing.T) {
	result := truncate("", 10)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestTruncate_ZeroMaxLen(t *testing.T) {
	result := truncate("hello", 0)
	expected := "..."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// ---------------------------------------------------------------------------
// paymentError is a test helper that implements the error interface
// ---------------------------------------------------------------------------

type paymentError struct {
	msg string
}

func (e *paymentError) Error() string {
	return e.msg
}

// ---------------------------------------------------------------------------
// Benchmark: VerifyWebhookSignature
// ---------------------------------------------------------------------------

func BenchmarkVerifyWebhookSignature_Valid(b *testing.B) {
	secret := "bench_secret_key_for_hmac_testing"
	svc := &paymentService{
		webhookSecret: secret,
	}

	payload := []byte(`{"event":"payment.captured","id":"evt_bench","payload":{"payment":{"entity":{"id":"pay_bench","email":"bench@test.com","amount":100000,"currency":"INR","notes":{"sponsor_name":"Benchmark User"}}}}}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.VerifyWebhookSignature(payload, signature)
	}
}

func BenchmarkVerifyWebhookSignature_Invalid(b *testing.B) {
	secret := "bench_secret_key_for_hmac_testing"
	svc := &paymentService{
		webhookSecret: secret,
	}

	payload := []byte(`{"event":"payment.captured","id":"evt_bench"}`)
	wrongSig := "0000000000000000000000000000000000000000000000000000000000000000"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.VerifyWebhookSignature(payload, wrongSig)
	}
}
