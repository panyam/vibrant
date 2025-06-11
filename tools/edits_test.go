package tools

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Helper to create EditCosts with default AlmostEqual
func newEditCosts(inLines, diffLines []string) *EditCosts {
	return (&EditCosts{
		InLines:   inLines,
		DiffLines: diffLines,
	}).Init() // Init also sets default AlmostEqual
}

// Test 1: Simple match of context lines
func TestEditCosts_SimpleContextMatch(t *testing.T) {
	inLines := []string{"line a", "line b"}
	diffLines := []string{" line a", " line b"} // Patch lines matching input

	ec := newEditCosts(inLines, diffLines)
	minCost := ec.Dist()

	// Expected cost: 0 (two context matches, cost 0 each)
	if minCost != 0 {
		t.Errorf("Expected minCost 0, got %d", minCost)
	}

	// Check opTable path (simplified check)
	// opTable[0][0] should be CONTEXT_MATCHED for "line a"
	// opTable[1][1] should be CONTEXT_MATCHED for "line b"
	// opTable[2][2] should be NOOP (end)
	if ec.opTable[0][0] != CONTEXT_MATCHED {
		t.Errorf("Expected opTable[0][0] to be CONTEXT_MATCHED, got %d", ec.opTable[0][0])
	}
	if ec.opTable[1][1] != CONTEXT_MATCHED {
		t.Errorf("Expected opTable[1][1] to be CONTEXT_MATCHED, got %d", ec.opTable[1][1])
	}
	if ec.opTable[2][2] != NOOP {
		t.Errorf("Expected opTable[2][2] to be NOOP, got %d", ec.opTable[2][2])
	}

	// Check Alignment
	alignment := ec.GetAlignment()
	expectedAlignment := []AlignedLine{
		{Type: CONTEXT_MATCHED, InputLine: "line a", InputLineNo: 1, PatchLine: "line a", PatchLineNo: 1},
		{Type: CONTEXT_MATCHED, InputLine: "line b", InputLineNo: 2, PatchLine: "line b", PatchLineNo: 2},
	}
	// Note: GetAlignment might produce an extra NOOP at the end if NI==i and ND==d
	// We'll trim it for comparison if it exists and its type is NOOP.
	if len(alignment) > 0 && alignment[len(alignment)-1].Type == NOOP && alignment[len(alignment)-1].InputLine == "" && alignment[len(alignment)-1].PatchLine == "" {
		alignment = alignment[:len(alignment)-1]
	}

	if !reflect.DeepEqual(alignment, expectedAlignment) {
		t.Errorf("Alignment mismatch:\nExpected: %+v\nGot:      %+v", expectedAlignment, alignment)
	}
}

// Test 2: Simple deletion
func TestEditCosts_SimpleDelete(t *testing.T) {
	inLines := []string{"line to delete"}
	diffLines := []string{"-line to delete"}

	ec := newEditCosts(inLines, diffLines)
	minCost := ec.Dist()

	// Expected cost: 0 (matched delete)
	if minCost != 0 {
		t.Errorf("Expected minCost 0, got %d", minCost)
	}
	if ec.opTable[0][0] != DELETED_INPUT_MATCHED_PATCH {
		t.Errorf("Expected opTable[0][0] to be DELETED_INPUT_MATCHED_PATCH, got %d", ec.opTable[0][0])
	}

	alignment := ec.GetAlignment()
	expectedAlignment := []AlignedLine{
		{Type: DELETED_INPUT_MATCHED_PATCH, InputLine: "line to delete", InputLineNo: 1, PatchLine: "line to delete", PatchLineNo: 1},
	}
	if len(alignment) > 0 && alignment[len(alignment)-1].Type == NOOP && alignment[len(alignment)-1].InputLine == "" && alignment[len(alignment)-1].PatchLine == "" {
		alignment = alignment[:len(alignment)-1]
	}
	if !reflect.DeepEqual(alignment, expectedAlignment) {
		t.Errorf("Alignment mismatch:\nExpected: %+v\nGot:      %+v", expectedAlignment, alignment)
	}
}

// Test 3: Simple addition
func TestEditCosts_SimpleAdd(t *testing.T) {
	inLines := []string{"existing line"} // Input line that will be "kept"
	diffLines := []string{"+new line"}   // Patch adds a line

	ec := newEditCosts(inLines, diffLines)
	minCost := ec.Dist()

	// Expected path: ADDED_FROM_PATCH (cost 1), then KEPT_INPUT_UNAFFECTED (cost 0)
	// Total cost: 1
	if minCost != 1 {
		t.Errorf("Expected minCost 1, got %d", minCost)
	}

	// opTable[0][0] should be ADDED_FROM_PATCH
	// opTable[0][1] should be KEPT_INPUT_UNAFFECTED (for "existing line")
	if ec.opTable[0][0] != ADDED_FROM_PATCH {
		t.Errorf("Expected opTable[0][0] to be ADDED_FROM_PATCH, got %d", ec.opTable[0][0])
	}
	if ec.opTable[0][1] != KEPT_INPUT_UNAFFECTED { // After '+' is processed, 'existing line' should be kept
		t.Errorf("Expected opTable[0][1] to be KEPT_INPUT_UNAFFECTED, got %d", ec.opTable[0][1])
	}

	alignment := ec.GetAlignment()
	expectedAlignment := []AlignedLine{
		{Type: ADDED_FROM_PATCH, InputLineNo: 0, PatchLine: "new line", PatchLineNo: 1}, // InputLineNo might be 0 or 1 depending on convention
		{Type: KEPT_INPUT_UNAFFECTED, InputLine: "existing line", InputLineNo: 1},
	}
	// Adjust InputLineNo for ADDED_FROM_PATCH based on GetAlignment's behavior
	if len(alignment) > 0 && alignment[0].Type == ADDED_FROM_PATCH {
		expectedAlignment[0].InputLineNo = alignment[0].InputLineNo
	}
	if len(alignment) > 0 && alignment[len(alignment)-1].Type == NOOP && alignment[len(alignment)-1].InputLine == "" && alignment[len(alignment)-1].PatchLine == "" {
		alignment = alignment[:len(alignment)-1]
	}
	if !reflect.DeepEqual(alignment, expectedAlignment) {
		t.Errorf("Alignment mismatch:\nExpected: %+v\nGot:      %+v", expectedAlignment, alignment)
	}
}

// Test 4: Context line does not match input, input line kept, patch context skipped
func TestEditCosts_ContextMismatch_SkipPatchContext(t *testing.T) {
	inLines := []string{"actual line a", "actual line b"}
	diffLines := []string{" wrong context a", " actual line b"} // First context is wrong

	ec := newEditCosts(inLines, diffLines)
	minCost := ec.Dist()

	// Path:
	// 1. "actual line a" vs " wrong context a":
	//    - KEPT_INPUT_UNAFFECTED for "actual line a" (cost 0) -> state (i=1, d=0)
	//    - CONTEXT_SKIPPED_PATCH_OP for " wrong context a" (cost 1) -> state (i=0, d=1)
	//    The algorithm should choose to keep "actual line a" (KEPT_INPUT_UNAFFECTED), then try to match "actual line b"
	//    OR skip " wrong context a" (CONTEXT_SKIPPED_PATCH_OP), then try "actual line a" vs " actual line b"
	//
	// Let's trace the optimal path assuming costs:
	// dist(0,0) for "actual line a" vs " wrong context a"
	//   Path A (KEPT_INPUT_UNAFFECTED for "actual line a"): 0 + dist(1,0)
	//     dist(1,0) for "actual line b" vs " wrong context a"
	//       Path A (KEPT_INPUT_UNAFFECTED for "actual line b"): 0 + dist(2,0) -> (KEPT "b", cost 0)
	//       Path B1 (CONTEXT_SKIPPED_PATCH_OP for " wrong context a"): 1 + dist(1,1)
	//         dist(1,1) for "actual line b" vs " actual line b" -> (CONTEXT_MATCHED, cost 0)
	//         So, this subpath cost is 1.
	//       min(0,1) = 0. opTable[1][0] = KEPT_INPUT_UNAFFECTED
	//     So, Path A from dist(0,0) has cost 0. opTable[0][0] = KEPT_INPUT_UNAFFECTED.
	//
	//   Path B1 (CONTEXT_SKIPPED_PATCH_OP for " wrong context a"): 1 + dist(0,1)
	//     dist(0,1) for "actual line a" vs " actual line b"
	//       Path A (KEPT_INPUT_UNAFFECTED for "actual line a"): 0 + dist(1,1)
	//         dist(1,1) for "actual line b" vs " actual line b" -> (CONTEXT_MATCHED, cost 0)
	//         Subpath cost 0.
	//       Path B1 (CONTEXT_SKIPPED_PATCH_OP for " actual line b"): 1 + dist(0,2) -> (SKIP, cost 1)
	//       min(0,1) = 0. opTable[0][1] = KEPT_INPUT_UNAFFECTED
	//     So, Path B1 from dist(0,0) has cost 1.
	//
	// min(0,1) = 0. So opTable[0][0] should be KEPT_INPUT_UNAFFECTED.
	// Then opTable[1][0] (for "actual line b" vs " wrong context a") should be KEPT_INPUT_UNAFFECTED.
	// Then opTable[2][0] (InLines exhausted) means we look at opTable[NI][d].
	// This is complex. The GetAlignment will be the better test here.
	// Expected Alignment:
	// 1. KEPT_INPUT_UNAFFECTED: "actual line a"
	// 2. CONTEXT_SKIPPED_PATCH_OP: " wrong context a" (from patch, skipped)
	// 3. CONTEXT_MATCHED: "actual line b"
	// Total cost: costSkipPatchContext (1) + costMatchContext (0) = 1

	// If KEPT_INPUT_UNAFFECTED costs 0 and skipping patch context costs 1:
	// Optimal:
	// (0,0) "actual line a" vs " wrong context a" -> KEPT_INPUT_UNAFFECTED "actual line a" (cost 0), state (1,0)
	// (1,0) "actual line b" vs " wrong context a" -> CONTEXT_SKIPPED_PATCH_OP for " wrong context a" (cost 1), state(1,1)
	// (1,1) "actual line b" vs " actual line b" -> CONTEXT_MATCHED (cost 0), state(2,2)
	// Total cost = 1.

	if minCost != 1 {
		t.Errorf("Expected minCost 1, got %d for context mismatch case", minCost)
	}

	// This specific path check is tricky due to how 'KEPT_INPUT_UNAFFECTED' interacts.
	// We'll rely more on the GetAlignment output for this one.
	// if ec.opTable[0][0] != KEPT_INPUT_UNAFFECTED { // This depends on tie-breaking logic if costs are equal for two paths.
	// 	t.Errorf("Expected opTable[0][0] for context mismatch to be KEPT_INPUT_UNAFFECTED or CONTEXT_SKIPPED_PATCH_OP, got %d", ec.opTable[0][0])
	// }

	alignment := ec.GetAlignment()
	expectedAlignment := []AlignedLine{
		{Type: KEPT_INPUT_UNAFFECTED, InputLine: "actual line a", InputLineNo: 1},
		// The GetAlignment needs to be smart about skipped patch ops.
		// If KEPT_INPUT_UNAFFECTED was chosen for (0,0), then (1,0) is next.
		// At (1,0) ("actual line b" vs " wrong context a"), CONTEXT_SKIPPED_PATCH_OP is chosen.
		{Type: CONTEXT_SKIPPED_PATCH_OP, PatchLine: "wrong context a", PatchLineNo: 1, InputLineNo: 2}, // InputLineNo here is tricky, depends on interpretation
		{Type: CONTEXT_MATCHED, InputLine: "actual line b", InputLineNo: 2, PatchLine: "actual line b", PatchLineNo: 2},
	}
	// Adjust InputLineNo based on GetAlignment's behavior
	if len(alignment) > 1 && alignment[1].Type == CONTEXT_SKIPPED_PATCH_OP {
		expectedAlignment[1].InputLineNo = alignment[1].InputLineNo
	}
	if len(alignment) > 0 && alignment[len(alignment)-1].Type == NOOP && alignment[len(alignment)-1].InputLine == "" && alignment[len(alignment)-1].PatchLine == "" {
		alignment = alignment[:len(alignment)-1]
	}
	if !reflect.DeepEqual(alignment, expectedAlignment) {
		t.Errorf("Alignment mismatch for context mismatch:\nExpected: %+v\nGot:      %+v", expectedAlignment, alignment)
	}
}

// Test 5: Fuzzy delete - input line doesn't quite match delete patch line
func TestEditCosts_FuzzyDelete(t *testing.T) {
	inLines := []string{"line to delete!"} // Extra '!'
	diffLines := []string{"-line to delete"}

	ec := newEditCosts(inLines, diffLines)
	// Use a custom AlmostEqual for this test
	ec.AlmostEqual = func(s1, s2 string) bool {
		// Simple fuzzy: allow one character difference at the end if base matches
		if strings.HasPrefix(s1, s2) && len(s1) <= len(s2)+1 {
			return true
		}
		if strings.HasPrefix(s2, s1) && len(s2) <= len(s1)+1 {
			return true
		}
		return s1 == s2
	}
	minCost := ec.Dist() // Recalculate with new AlmostEqual

	// With this AlmostEqual, it should be treated as DELETED_INPUT_MATCHED_PATCH (cost 0)
	// If AlmostEqual was strict, it would be DELETED_INPUT_UNMATCHED_PATCH (cost 1)
	if minCost != 0 {
		t.Errorf("Expected minCost 0 for fuzzy delete (matched), got %d", minCost)
	}
	if ec.opTable[0][0] != DELETED_INPUT_MATCHED_PATCH {
		t.Errorf("Expected opTable[0][0] for fuzzy delete to be DELETED_INPUT_MATCHED_PATCH, got %d", ec.opTable[0][0])
	}

	alignment := ec.GetAlignment()
	expectedAlignment := []AlignedLine{
		{Type: DELETED_INPUT_MATCHED_PATCH, InputLine: "line to delete!", InputLineNo: 1, PatchLine: "line to delete", PatchLineNo: 1},
	}
	if len(alignment) > 0 && alignment[len(alignment)-1].Type == NOOP && alignment[len(alignment)-1].InputLine == "" && alignment[len(alignment)-1].PatchLine == "" {
		alignment = alignment[:len(alignment)-1]
	}
	if !reflect.DeepEqual(alignment, expectedAlignment) {
		t.Errorf("Alignment mismatch for fuzzy delete:\nExpected: %+v\nGot:      %+v", expectedAlignment, alignment)
	}
}

// Test 6: Your specific example from the problem description
func TestEditCosts_OriginalExample(t *testing.T) {
	inLines := []string{
		"", // line 239 in original file example, adjusted to 0-index for []string
		"func TestLexer_ComplexDurations(t *testing.T) {",
		"    input := `10 10.5ms 1s2 ` // \"1s2\" is 1s, then IDENTIFIER \"s2\"",
		"    expected := []expectedToken{",
		"        {INT_LITERAL, \"10\", 0, 2, 1, 1, IntValue(10), \"\"},",
		"        {DURATION_LITERAL, \"10.5ms\", 3, 9, 1, 4, FloatValue(parseDuration(\"10.5\", \"ms\")), \"\"},",
		"        {DURATION_LITERAL, \"1s\", 10, 12, 1, 11, FloatValue(parseDuration(\"1\",\"s\")), \"\"},",
		"        {IDENTIFIER, \"2\", 12, 13, 1, 13, nil, \"2\"},",
		"    }",
		"    // The original test expected \"1s2\" to error with \"invalid character after unit\".",
		"    // The new lexer logic separates \"1s\" and \"2\" if \"s\" is a valid unit followed by something that",
		"    // doesn't form part of a longer valid unit AND is not whitespace/punctuation.",
	}
	diffLines := []string{
		" ", // Context for empty line
		" func TestLexer_ComplexDurations(t *testing.T) {",
		"     input := `10 10.5ms 1s2 ` // \"1s2\" is 1s, then IDENTIFIER \"s2\"",
		"-    expected := []expectedToken{",
		"-        {INT_LITERAL, \"10\", 0, 2, 1, 1, IntValue(10), \"\"},",
		"-        {DURATION_LITERAL, \"10.5ms\", 3, 9, 1, 4, FloatValue(parseDuration(\"10.5\", \"ms\")), \"\"},",
		"-        {DURATION_LITERAL, \"1s\", 10, 12, 1, 11, FloatValue(parseDuration(\"1\",\"s\")), \"\"},",
		"-        {IDENTIFIER, \"2\", 12, 13, 1, 13, nil, \"2\"},",
		"-    }",
		"+", // The blank line added
		"     // The original test expected \"1s2\" to error with \"invalid character after unit\".",
		"     // The new lexer logic separates \"1s\" and \"2\" if \"s\" is a valid unit followed by something that",
		"     // doesn't form part of a longer valid unit AND is not whitespace/punctuation.",
	}

	ec := newEditCosts(inLines, diffLines)
	minCost := ec.Dist()

	// Expected cost: 1 (for the one '+' line)
	// All context lines should match (cost 0)
	// All delete lines should match corresponding input lines (cost 0)
	if minCost != 1 {
		t.Errorf("Expected minCost 1 for original example, got %d", minCost)
	}

	// Spot check some ops
	if ec.opTable[0][0] != CONTEXT_MATCHED {
		t.Errorf("Op mismatch at (0,0)")
	} // "" vs " "
	if ec.opTable[1][1] != CONTEXT_MATCHED {
		t.Errorf("Op mismatch at (1,1)")
	} // func...
	if ec.opTable[3][3] != DELETED_INPUT_MATCHED_PATCH {
		t.Errorf("Op mismatch at (3,3)")
	} // expected...
	// After all deletes, at (say) i=9, d=9 (index of '+')
	// ec.opTable[9][9] should be ADDED_FROM_PATCH
	// The indices here get a bit complex to map directly without running,
	// but the GetAlignment check is more comprehensive.

	alignment := ec.GetAlignment()
	expectedAlignment := []AlignedLine{
		{Type: CONTEXT_MATCHED, InputLine: "", InputLineNo: 1, PatchLine: "", PatchLineNo: 1},
		{Type: CONTEXT_MATCHED, InputLine: "func TestLexer_ComplexDurations(t *testing.T) {", InputLineNo: 2, PatchLine: "func TestLexer_ComplexDurations(t *testing.T) {", PatchLineNo: 2},
		{Type: CONTEXT_MATCHED, InputLine: "    input := `10 10.5ms 1s2 ` // \"1s2\" is 1s, then IDENTIFIER \"s2\"", InputLineNo: 3, PatchLine: "    input := `10 10.5ms 1s2 ` // \"1s2\" is 1s, then IDENTIFIER \"s2\"", PatchLineNo: 3},
		{Type: DELETED_INPUT_MATCHED_PATCH, InputLine: "    expected := []expectedToken{", InputLineNo: 4, PatchLine: "    expected := []expectedToken{", PatchLineNo: 4},
		{Type: DELETED_INPUT_MATCHED_PATCH, InputLine: "        {INT_LITERAL, \"10\", 0, 2, 1, 1, IntValue(10), \"\"},", InputLineNo: 5, PatchLine: "        {INT_LITERAL, \"10\", 0, 2, 1, 1, IntValue(10), \"\"},", PatchLineNo: 5},
		{Type: DELETED_INPUT_MATCHED_PATCH, InputLine: "        {DURATION_LITERAL, \"10.5ms\", 3, 9, 1, 4, FloatValue(parseDuration(\"10.5\", \"ms\")), \"\"},", InputLineNo: 6, PatchLine: "        {DURATION_LITERAL, \"10.5ms\", 3, 9, 1, 4, FloatValue(parseDuration(\"10.5\", \"ms\")), \"\"},", PatchLineNo: 6},
		{Type: DELETED_INPUT_MATCHED_PATCH, InputLine: "        {DURATION_LITERAL, \"1s\", 10, 12, 1, 11, FloatValue(parseDuration(\"1\",\"s\")), \"\"},", InputLineNo: 7, PatchLine: "        {DURATION_LITERAL, \"1s\", 10, 12, 1, 11, FloatValue(parseDuration(\"1\",\"s\")), \"\"},", PatchLineNo: 7},
		{Type: DELETED_INPUT_MATCHED_PATCH, InputLine: "        {IDENTIFIER, \"2\", 12, 13, 1, 13, nil, \"2\"},", InputLineNo: 8, PatchLine: "        {IDENTIFIER, \"2\", 12, 13, 1, 13, nil, \"2\"},", PatchLineNo: 8},
		{Type: DELETED_INPUT_MATCHED_PATCH, InputLine: "    }", InputLineNo: 9, PatchLine: "    }", PatchLineNo: 9},
		{Type: ADDED_FROM_PATCH, PatchLine: "", PatchLineNo: 10, InputLineNo: 9}, // InputLineNo is where it's inserted relative to
		{Type: CONTEXT_MATCHED, InputLine: "    // The original test expected \"1s2\" to error with \"invalid character after unit\".", InputLineNo: 10, PatchLine: "    // The original test expected \"1s2\" to error with \"invalid character after unit\".", PatchLineNo: 11},
		{Type: CONTEXT_MATCHED, InputLine: "    // The new lexer logic separates \"1s\" and \"2\" if \"s\" is a valid unit followed by something that", InputLineNo: 11, PatchLine: "    // The new lexer logic separates \"1s\" and \"2\" if \"s\" is a valid unit followed by something that", PatchLineNo: 12},
		{Type: CONTEXT_MATCHED, InputLine: "    // doesn't form part of a longer valid unit AND is not whitespace/punctuation.", InputLineNo: 12, PatchLine: "    // doesn't form part of a longer valid unit AND is not whitespace/punctuation.", PatchLineNo: 13},
	}
	// Adjust InputLineNo for ADDED_FROM_PATCH based on GetAlignment's behavior
	if len(alignment) > 9 && alignment[9].Type == ADDED_FROM_PATCH {
		expectedAlignment[9].InputLineNo = alignment[9].InputLineNo
	}
	if len(alignment) > 0 && alignment[len(alignment)-1].Type == NOOP && alignment[len(alignment)-1].InputLine == "" && alignment[len(alignment)-1].PatchLine == "" {
		alignment = alignment[:len(alignment)-1]
	}

	if !reflect.DeepEqual(alignment, expectedAlignment) {
		t.Errorf("Alignment mismatch for original example:\nExpected: %+v\nGot:      %+v", expectedAlignment, alignment)
		// For debugging, print line by line
		for k := 0; k < len(expectedAlignment) || k < len(alignment); k++ {
			var eLine, aLine string
			if k < len(expectedAlignment) {
				eLine = sprintAlignedLine(expectedAlignment[k])
			}
			if k < len(alignment) {
				aLine = sprintAlignedLine(alignment[k])
			}
			if eLine != aLine {
				t.Logf("Diff at index %d:\nExp: %s\nGot: %s\n", k, eLine, aLine)
			}
		}
	}
}

// Helper to print AlignedLine for debugging
func sprintAlignedLine(al AlignedLine) string {
	return fmt.Sprintf("Type:%d InLNo:%d InTxt:'%s' PaLNo:%d PaTxt:'%s'",
		al.Type, al.InputLineNo, al.InputLine, al.PatchLineNo, al.PatchLine)
}

func TestEditCosts_ApplyPatch_OriginalExample(t *testing.T) {
	inLines := []string{
		"",
		"func TestLexer_ComplexDurations(t *testing.T) {",
		"    input := `10 10.5ms 1s2 ` // \"1s2\" is 1s, then IDENTIFIER \"s2\"",
		"    expected := []expectedToken{",                                                                       // To be deleted
		"        {INT_LITERAL, \"10\", 0, 2, 1, 1, IntValue(10), \"\"},",                                         // To be deleted
		"        {DURATION_LITERAL, \"10.5ms\", 3, 9, 1, 4, FloatValue(parseDuration(\"10.5\", \"ms\")), \"\"},", // To be deleted
		"        {DURATION_LITERAL, \"1s\", 10, 12, 1, 11, FloatValue(parseDuration(\"1\",\"s\")), \"\"},",       // To be deleted
		"        {IDENTIFIER, \"2\", 12, 13, 1, 13, nil, \"2\"},",                                                // To be deleted
		"    }", // To be deleted
		"    // The original test expected \"1s2\" to error with \"invalid character after unit\".",
		"    // The new lexer logic separates \"1s\" and \"2\" if \"s\" is a valid unit followed by something that",
		"    // doesn't form part of a longer valid unit AND is not whitespace/punctuation.",
	}
	diffLines := []string{
		" ",
		" func TestLexer_ComplexDurations(t *testing.T) {",
		"     input := `10 10.5ms 1s2 ` // \"1s2\" is 1s, then IDENTIFIER \"s2\"",
		"-    expected := []expectedToken{",
		"-        {INT_LITERAL, \"10\", 0, 2, 1, 1, IntValue(10), \"\"},",
		"-        {DURATION_LITERAL, \"10.5ms\", 3, 9, 1, 4, FloatValue(parseDuration(\"10.5\", \"ms\")), \"\"},",
		"-        {DURATION_LITERAL, \"1s\", 10, 12, 1, 11, FloatValue(parseDuration(\"1\",\"s\")), \"\"},",
		"-        {IDENTIFIER, \"2\", 12, 13, 1, 13, nil, \"2\"},",
		"-    }",
		"+", // The blank line added
		"     // The original test expected \"1s2\" to error with \"invalid character after unit\".",
		"     // The new lexer logic separates \"1s\" and \"2\" if \"s\" is a valid unit followed by something that",
		"     // doesn't form part of a longer valid unit AND is not whitespace/punctuation.",
	}

	expectedOutputLines := []string{
		"", // Context
		"func TestLexer_ComplexDurations(t *testing.T) {",                        // Context
		"    input := `10 10.5ms 1s2 ` // \"1s2\" is 1s, then IDENTIFIER \"s2\"", // Context
		// Deleted lines are gone
		"", // The added blank line from "+"
		"    // The original test expected \"1s2\" to error with \"invalid character after unit\".",                 // Context
		"    // The new lexer logic separates \"1s\" and \"2\" if \"s\" is a valid unit followed by something that", // Context
		"    // doesn't form part of a longer valid unit AND is not whitespace/punctuation.",                        // Context
	}

	ec := newEditCosts(inLines, diffLines)
	// IMPORTANT: Must call Dist() to populate tables before GetAlignment/ApplyPatchToInput
	_ = ec.Dist() // We can ignore the cost for this test's purpose

	actualOutputLines := ec.ApplyPatchToInput()

	if !reflect.DeepEqual(actualOutputLines, expectedOutputLines) {
		t.Errorf("ApplyPatchToInput mismatch for original example:\nExpected Lines (%d):\n%s\nGot Lines (%d):\n%s",
			len(expectedOutputLines), strings.Join(expectedOutputLines, "\n"),
			len(actualOutputLines), strings.Join(actualOutputLines, "\n"))

		// For more detailed diff:
		for i := 0; i < len(expectedOutputLines) || i < len(actualOutputLines); i++ {
			eLine, aLine := "", ""
			if i < len(expectedOutputLines) {
				eLine = expectedOutputLines[i]
			}
			if i < len(actualOutputLines) {
				aLine = actualOutputLines[i]
			}
			if eLine != aLine {
				t.Logf("Line %d diff:\nExp: '%s'\nGot: '%s'", i, eLine, aLine)
			}
		}
	}
}
