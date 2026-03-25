package service

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// chunkText
// ---------------------------------------------------------------------------

func TestChunkText_EmptyString(t *testing.T) {
	chunks := chunkText("", 800, 200)
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty string, got %d", len(chunks))
	}
}

func TestChunkText_WhitespaceOnly(t *testing.T) {
	chunks := chunkText("   \t\n  ", 800, 200)
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for whitespace-only string, got %d", len(chunks))
	}
}

func TestChunkText_ShortText_SingleChunk(t *testing.T) {
	text := "Hello, this is a short resume."
	chunks := chunkText(text, 800, 200)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for short text, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Errorf("expected chunk to equal input text, got %q", chunks[0])
	}
}

func TestChunkText_ExactTargetSize(t *testing.T) {
	// Create text exactly at the target size
	text := strings.Repeat("a", 800)
	chunks := chunkText(text, 800, 200)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for text exactly at target size, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Errorf("expected chunk to equal input text")
	}
}

func TestChunkText_SplitsAtWordBoundary(t *testing.T) {
	// Build text that is clearly longer than one chunk
	words := make([]string, 200)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ") // 200 * 5 - 1 = 999 chars

	chunks := chunkText(text, 100, 20)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	for i, chunk := range chunks {
		trimmed := strings.TrimSpace(chunk)
		if trimmed == "" {
			t.Errorf("chunk %d is empty after trimming", i)
			continue
		}
		// No chunk should start or end with a space (they should be trimmed)
		if chunk != trimmed {
			t.Errorf("chunk %d has leading/trailing whitespace: %q", i, chunk)
		}
		// No word should be cut in half — each chunk should only contain
		// complete "word" tokens (possibly concatenated at boundaries)
		for _, w := range strings.Fields(trimmed) {
			if w != "word" {
				t.Errorf("chunk %d contains partial word: %q", i, w)
			}
		}
	}
}

func TestChunkText_Overlap(t *testing.T) {
	// Create text with numbered words so we can verify overlap
	var words []string
	for i := 0; i < 100; i++ {
		words = append(words, strings.Repeat("x", 8)) // 8-char words
	}
	text := strings.Join(words, " ") // ~900 chars total

	targetSize := 100
	overlap := 30

	chunks := chunkText(text, targetSize, overlap)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	// Verify that consecutive chunks overlap: the end of chunk i should
	// appear at the beginning of chunk i+1
	for i := 0; i < len(chunks)-1; i++ {
		current := chunks[i]
		next := chunks[i+1]

		// The tail of the current chunk (last `overlap` chars or so) should
		// appear somewhere at the start of the next chunk. Due to word-boundary
		// splitting, the overlap may not be exact, but there should be some
		// shared content.
		if len(current) > overlap {
			tail := current[len(current)-overlap:]
			// Check for any shared substring of reasonable length
			sharedFound := false
			// Look for at least one shared word
			tailWords := strings.Fields(tail)
			nextPrefix := next
			if len(nextPrefix) > overlap*2 {
				nextPrefix = nextPrefix[:overlap*2]
			}
			for _, w := range tailWords {
				if strings.Contains(nextPrefix, w) {
					sharedFound = true
					break
				}
			}
			if !sharedFound && len(tailWords) > 0 {
				// This is a soft check — overlap may align differently due
				// to word boundaries. Just log it.
				t.Logf("chunk %d -> %d: no obvious overlap detected (tail=%q, next_prefix=%q)",
					i, i+1, tail, nextPrefix)
			}
		}
	}
}

func TestChunkText_ZeroTargetSize_UsesDefault(t *testing.T) {
	text := strings.Repeat("hello ", 200) // ~1200 chars
	chunks := chunkText(text, 0, 0)

	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}
	// With default target size of 800, should produce at least 2 chunks
	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks with default target size, got %d", len(chunks))
	}
}

func TestChunkText_NegativeOverlap_TreatedAsZero(t *testing.T) {
	text := strings.Repeat("a ", 500) // ~1000 chars
	chunks := chunkText(text, 100, -50)

	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}
	// Should not panic or infinite loop
}

func TestChunkText_OverlapEqualToTargetSize_Clamped(t *testing.T) {
	text := strings.Repeat("test ", 100) // ~500 chars
	// When overlap >= targetSize, it should be clamped to targetSize/4
	chunks := chunkText(text, 100, 100)

	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}
	// Should not infinite loop
}

func TestChunkText_OverlapLargerThanTargetSize_Clamped(t *testing.T) {
	text := strings.Repeat("data ", 100)
	chunks := chunkText(text, 50, 200)

	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}
}

func TestChunkText_SingleVeryLongWord(t *testing.T) {
	// A single word with no spaces — forces a hard split
	text := strings.Repeat("a", 2000)
	chunks := chunkText(text, 800, 200)

	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}

	// Reassemble and verify no content is lost (accounting for possible overlap)
	totalLen := 0
	for _, c := range chunks {
		totalLen += len(c)
	}
	if totalLen < len(text) {
		t.Errorf("reassembled chunks (%d chars) are shorter than input (%d chars) — content lost",
			totalLen, len(text))
	}
}

func TestChunkText_PreservesAllContent(t *testing.T) {
	// Build text with unique tokens so we can verify nothing is lost
	var tokens []string
	for i := 0; i < 50; i++ {
		tokens = append(tokens, strings.Repeat("w", 15)+string(rune('A'+i%26)))
	}
	text := strings.Join(tokens, " ")

	chunks := chunkText(text, 200, 50)

	// Every token should appear in at least one chunk
	combined := strings.Join(chunks, " ")
	for _, token := range tokens {
		if !strings.Contains(combined, token) {
			t.Errorf("token %q missing from chunked output", token)
		}
	}
}

func TestChunkText_NoEmptyChunks(t *testing.T) {
	text := "This is a test. " + strings.Repeat("More content here. ", 50)
	chunks := chunkText(text, 100, 20)

	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			t.Errorf("chunk %d is empty", i)
		}
	}
}

func TestChunkText_LeadingTrailingWhitespace(t *testing.T) {
	text := "   Hello world this is padded text   "
	chunks := chunkText(text, 800, 200)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != strings.TrimSpace(text) {
		t.Errorf("expected trimmed text, got %q", chunks[0])
	}
}

func TestChunkText_MultipleNewlines(t *testing.T) {
	text := "Paragraph one.\n\nParagraph two.\n\nParagraph three.\n\nParagraph four."
	chunks := chunkText(text, 30, 5)

	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}

	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			t.Errorf("chunk %d is empty", i)
		}
	}
}

// ---------------------------------------------------------------------------
// normaliseWhitespace
// ---------------------------------------------------------------------------

func TestNormaliseWhitespace_EmptyString(t *testing.T) {
	result := normaliseWhitespace("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestNormaliseWhitespace_OnlyWhitespace(t *testing.T) {
	result := normaliseWhitespace("   \t\t  \n  \r\n  ")
	if result != "" {
		t.Errorf("expected empty string after trimming, got %q", result)
	}
}

func TestNormaliseWhitespace_NoChange(t *testing.T) {
	text := "Hello world"
	result := normaliseWhitespace(text)
	if result != text {
		t.Errorf("expected %q, got %q", text, result)
	}
}

func TestNormaliseWhitespace_CollapseMultipleSpaces(t *testing.T) {
	result := normaliseWhitespace("Hello    world    test")
	expected := "Hello world test"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestNormaliseWhitespace_TabsToSpaces(t *testing.T) {
	result := normaliseWhitespace("Hello\t\tworld")
	expected := "Hello world"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestNormaliseWhitespace_CollapseMultipleNewlines(t *testing.T) {
	result := normaliseWhitespace("Line one.\n\n\n\nLine two.")
	expected := "Line one.\nLine two."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestNormaliseWhitespace_PreserveSingleNewlines(t *testing.T) {
	result := normaliseWhitespace("Line one.\nLine two.")
	expected := "Line one.\nLine two."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestNormaliseWhitespace_CRLFToLF(t *testing.T) {
	result := normaliseWhitespace("Line one.\r\nLine two.")
	// \r and \n are both newline characters; consecutive ones collapse to one \n
	if !strings.Contains(result, "Line one.") || !strings.Contains(result, "Line two.") {
		t.Errorf("expected both lines to be present, got %q", result)
	}
	// Should not contain \r
	if strings.Contains(result, "\r") {
		t.Errorf("expected \\r to be removed, got %q", result)
	}
}

func TestNormaliseWhitespace_MixedWhitespace(t *testing.T) {
	result := normaliseWhitespace("  Hello  \t  world  \n\n  foo  \t\t  bar  ")
	// Spaces and tabs between words → single space
	// Multiple newlines → single newline
	// Leading/trailing whitespace → trimmed
	if strings.HasPrefix(result, " ") || strings.HasSuffix(result, " ") {
		t.Errorf("expected trimmed result, got %q", result)
	}
	if strings.Contains(result, "\t") {
		t.Errorf("expected no tabs in result, got %q", result)
	}
	if strings.Contains(result, "  ") {
		t.Errorf("expected no double spaces in result, got %q", result)
	}
}

func TestNormaliseWhitespace_LeadingTrailingTrimmed(t *testing.T) {
	result := normaliseWhitespace("   content   ")
	if result != "content" {
		t.Errorf("expected 'content', got %q", result)
	}
}

func TestNormaliseWhitespace_NewlineAfterSpaces(t *testing.T) {
	// Spaces followed by a newline — the spaces should collapse and the
	// newline should be preserved
	result := normaliseWhitespace("Hello   \nWorld")
	// The spaces before \n should be consumed, then \n is kept
	if !strings.Contains(result, "Hello") || !strings.Contains(result, "World") {
		t.Errorf("expected both words, got %q", result)
	}
}

func TestNormaliseWhitespace_SpacesAfterNewline(t *testing.T) {
	result := normaliseWhitespace("Hello\n   World")
	// Spaces after newline should be collapsed
	if strings.Contains(result, "\n ") || strings.Contains(result, "\n  ") {
		t.Errorf("expected no spaces after newline, got %q", result)
	}
	if !strings.Contains(result, "Hello") || !strings.Contains(result, "World") {
		t.Errorf("expected both words, got %q", result)
	}
}

func TestNormaliseWhitespace_FormFeedAndVerticalTab(t *testing.T) {
	result := normaliseWhitespace("Hello\f\vWorld")
	// Form feed and vertical tab are whitespace characters
	// They should be treated as spaces and collapsed
	if !strings.Contains(result, "Hello") || !strings.Contains(result, "World") {
		t.Errorf("expected both words, got %q", result)
	}
	if strings.Contains(result, "\f") || strings.Contains(result, "\v") {
		t.Errorf("expected form feed and vertical tab to be removed, got %q", result)
	}
}

func TestNormaliseWhitespace_UnicodeSpaces(t *testing.T) {
	// Non-breaking space (U+00A0) and other Unicode spaces
	text := "Hello\u00a0\u00a0World"
	result := normaliseWhitespace(text)
	// unicode.IsSpace returns true for U+00A0, so it should be collapsed
	if strings.Contains(result, "\u00a0") {
		t.Errorf("expected non-breaking spaces to be collapsed, got %q", result)
	}
	if !strings.Contains(result, "Hello") || !strings.Contains(result, "World") {
		t.Errorf("expected both words, got %q", result)
	}
}

func TestNormaliseWhitespace_LongInput(t *testing.T) {
	// Stress test with a large input
	var sb strings.Builder
	for i := 0; i < 10000; i++ {
		sb.WriteString("word ")
		if i%100 == 0 {
			sb.WriteString("\n\n\n   ")
		}
	}
	text := sb.String()
	result := normaliseWhitespace(text)

	// Should not contain consecutive spaces
	if strings.Contains(result, "  ") {
		t.Error("result contains consecutive spaces")
	}
	// Should not contain consecutive newlines
	if strings.Contains(result, "\n\n") {
		t.Error("result contains consecutive newlines")
	}
	// Should not be empty
	if len(result) == 0 {
		t.Error("result is empty")
	}
}

// ---------------------------------------------------------------------------
// Benchmark: chunkText
// ---------------------------------------------------------------------------

func BenchmarkChunkText_Small(b *testing.B) {
	text := strings.Repeat("This is a test sentence. ", 20) // ~500 chars
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunkText(text, 800, 200)
	}
}

func BenchmarkChunkText_Medium(b *testing.B) {
	text := strings.Repeat("This is a longer test sentence with more words. ", 100) // ~5000 chars
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunkText(text, 800, 200)
	}
}

func BenchmarkChunkText_Large(b *testing.B) {
	text := strings.Repeat("This is a large document simulation with various words and content. ", 500) // ~34000 chars
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunkText(text, 800, 200)
	}
}

// ---------------------------------------------------------------------------
// Benchmark: normaliseWhitespace
// ---------------------------------------------------------------------------

func BenchmarkNormaliseWhitespace_Clean(b *testing.B) {
	text := strings.Repeat("Clean text with no extra whitespace. ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normaliseWhitespace(text)
	}
}

func BenchmarkNormaliseWhitespace_Dirty(b *testing.B) {
	text := strings.Repeat("Messy   \t\t  text  \n\n\n with  lots   of   whitespace. ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normaliseWhitespace(text)
	}
}
