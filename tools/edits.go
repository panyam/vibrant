package tools

import (
	"strings"

	"github.com/panyam/goutils/fn"
)

/**
---

**`EditCosts` Algorithm: Fuzzy Patch Alignment**

**Purpose:**
The `EditCosts` algorithm provides a mechanism to align a sequence of input text lines (`InLines`) with a sequence of patch/diff lines (`DiffLines`), especially when the patch might be slightly corrupted, outdated, or not perfectly applicable using standard tools. Its primary goal is to produce a "good enough" alignment that can be used to:
1.  Generate an annotated, interleaved view of changes (showing original lines, added lines, deleted lines, and context).
2.  Produce the resulting text lines as if the patch operations were applied.

This algorithm serves as a fallback when standard `patch` utilities (even with fuzz factors) fail or when a more detailed, step-by-step understanding of the patch application is needed.

**Core Technique:**
The algorithm is based on **dynamic programming**, similar in principle to global sequence alignment algorithms like Needleman-Wunsch. However, it's specialized to understand the semantics of patch file lines (context `' '`, additions `'+'`, deletions `'-'`).

It constructs a 2D table (`costTable`) where `costTable[i][d]` stores the minimum "edit cost" to align the first `i` lines of `InLines` with the first `d` lines of `DiffLines`. A corresponding `opTable[i][d]` stores the optimal operation that leads to this minimum cost.

**Key Components:**

1.  **`InLines`**: A slice of strings representing the original input file content.
2.  **`DiffLines`**: A slice of strings representing the lines from a patch hunk (e.g., `" line context"`, `"+line to add"`, `"-line to delete"`).
3.  **`AlmostEqual` Function**: A pluggable function `func(s1, s2 string) bool` that determines if two lines are considered "close enough" for a match (e.g., exact match, ignoring whitespace, Levenshtein distance below a threshold). This enables fuzzy matching.
4.  **Cost Model**: The algorithm's behavior is guided by a set of costs associated with different operations:
    *   Matching a context line (`' '`) from `DiffLines` with an `InLines` line: Low cost (e.g., 0).
    *   Matching a delete line (`'-'`) from `DiffLines` with an `InLines` line: Low cost (e.g., 0).
    *   Applying an add line (`'+'`) from `DiffLines`: Moderate cost (e.g., 1).
    *   Keeping an `InLines` line that is not directly affected by the current `DiffLines` operation: Low cost (e.g., 0).
    *   Skipping a patch operation (`' '` or `'-'`) if it doesn't match the current `InLines` line: Moderate cost (e.g., 1).
    *   Forcing a delete operation on an `InLines` line even if it doesn't perfectly match the `DiffLines` delete instruction: Moderate to high cost (e.g., 1 or more).
    The specific costs are tunable and dictate the algorithm's preferences when ambiguities arise.
5.  **Operation Codes (`opTable`)**: The `opTable` stores codes representing the decision made at each step of the alignment. These codes are more descriptive than simple add/delete/change and reflect the patch context:
    *   `KEPT_INPUT_UNAFFECTED`: An input line was kept as is.
    *   `CONTEXT_MATCHED`: An input line matched a patch context line.
    *   `CONTEXT_SKIPPED_PATCH_OP`: A patch context line was skipped as it didn't match the current input line.
    *   `DELETED_INPUT_MATCHED_PATCH`: An input line matched a patch delete line and is considered deleted.
    *   `DELETED_INPUT_UNMATCHED_PATCH`: An input line (not matching a patch delete line) was still considered deleted (fuzzy application).
    *   `DELETE_SKIPPED_PATCH_OP`: A patch delete line was skipped.
    *   `ADDED_FROM_PATCH`: A line was added from a patch add line.
    *   `NOOP`: No operation, or skipping an invalid patch line.

**Workflow:**

1.  **Initialization (`Init`)**:
    *   Sets up the `costTable`, `opTable`, and `visited` (for memoization) tables.
    *   Initializes boundary conditions:
        *   If `InLines` are exhausted, remaining `DiffLines` are processed (e.g., `+` lines are added, `' '`/`'-'` lines are skipped, each with a cost).
        *   If `DiffLines` are exhausted, remaining `InLines` are kept (at low/no cost).
2.  **Distance Calculation (`dist(i, d)`)**:
    *   Recursively calculates `costTable[i][d]` and `opTable[i][d]`.
    *   For state `(i, d)` (considering `InLines[i]` and `DiffLines[d]`):
        *   It explores several "paths" or choices:
            1.  Keep `InLines[i]` (advancing `i` to `i+1`, `d` stays `d`).
            2.  If `DiffLines[d]` is context (`' '`):
                *   Try to match `InLines[i]` with `DiffLines[d]`. If `AlmostEqual`, advance both `i` and `d`.
                *   If not `AlmostEqual`, try skipping `DiffLines[d]` (advancing `d`, `i` stays `i`).
            3.  If `DiffLines[d]` is an add (`'+'`): Apply the addition (advancing `d`, `i` stays `i`).
            4.  If `DiffLines[d]` is a delete (`'-'`):
                *   Try to match `InLines[i]` for deletion. If `AlmostEqual`, advance both `i` and `d`.
                *   If not `AlmostEqual`, consider forcibly deleting `InLines[i]` (advancing `i` and `d`) or skipping the patch delete `DiffLines[d]` (advancing `d`, `i` stays `i`).
        *   The path with the minimum accumulated cost is chosen, and its cost and leading operation are stored.
    *   Memoization (`visited` table) prevents re-computation of subproblems.
3.  **Get Final Cost (`Dist`)**: Calls `Init()` then `dist(0,0)` to get the overall minimum cost.
4.  **Reconstruct Alignment (`GetAlignment`)**:
    *   Backtracks through the `opTable` from `(0,0)` to `(NI, ND)`.
    *   Generates a sequence of `AlignedLine` structs, each describing an operation (using the detailed op codes), the involved input line (if any), and the involved patch line (if any). This sequence provides the rich, annotated diff.
5.  **Apply Patch (`ApplyPatchToInput`)**:
    *   Uses the result of `GetAlignment()`.
    *   Constructs a new list of strings by applying the operations: keeping lines, adding lines from the patch, and omitting deleted lines. This produces the "patched file" content.

**Strengths:**
*   Handles fuzzy matches and slightly misaligned patches gracefully.
*   Provides a detailed trace of how the patch is interpreted against the input.
*   Tunable via the cost model and `AlmostEqual` function.

**Considerations for Future Development:**
*   **Cost Tuning**: The specific cost values are critical. Experimentation may be needed to achieve the most intuitive alignments for various scenarios.
*   **Tie-Breaking**: The logic for choosing an operation when multiple paths yield the same minimum cost can influence the output. Current tie-breaking generally prefers applying a patch operation over simply keeping an input line if costs are equal.
*   **`GetAlignment` Reconstruction**: The logic to reconstruct the `AlignedLine` sequence from the `opTable` needs to be robust, especially for "skip patch op" scenarios, to accurately reflect the alignment decisions for display purposes.
*   **Performance**: For very large files or very long patch hunks, the `O(NI * ND)` complexity of the DP algorithm could be a concern, though it's standard for such problems.

---
*/

type EditOp int

const (
	costMatchContext         = 0
	costKeepInputLine        = 0
	costSkipPatchContext     = 1
	costAddPatchLine         = 1
	costMatchDelete          = 0
	costFuzzyDelete          = 2
	costSkipPatchDelete      = 1
	costSkipInvalidPatchLine = 1
)

const (
	NOOP EditOp = iota // General placeholder, or could mean "skip patch line, keep input line pointer"

	// Operations related to InLines
	KEPT_INPUT_UNAFFECTED = 10 // InLines[i] kept, no specific patch op action on it here
	// (e.g., patch lines exhausted, or skipping a patch op)

	// Operations driven by matching patch context lines (' ')
	CONTEXT_MATCHED          = 20 // InLines[i] matched DiffLines[d] (context)
	CONTEXT_SKIPPED_PATCH_OP = 21 // InLines[i] kept, DiffLines[d] (context) didn't match InLines[i] and was skipped

	// Operations driven by patch delete lines ('-')
	DELETED_INPUT_MATCHED_PATCH   = 30 // InLines[i] matched DiffLines[d] (delete op) and is considered deleted
	DELETED_INPUT_UNMATCHED_PATCH = 31 // InLines[i] did NOT match DiffLines[d] (delete op), but InLines[i] is still considered deleted (forced delete)
	DELETE_SKIPPED_PATCH_OP       = 32 // InLines[i] kept, DiffLines[d] (delete op) didn't match InLines[i] and was skipped

	// Operations driven by patch add lines ('+')
	ADDED_FROM_PATCH = 40 // DiffLines[d] (add op) is inserted

	// Could add more if needed, e.g., for replace operations if you extend patch parsing
)

const (
	ADD1    = 1
	ADD2    = 2
	DEL1    = 3
	DEL2    = 4
	REPLACE = 5
	SAME    = 6
)

type AlignedLine struct {
	Type        EditOp // One of your new op codes
	InputLine   string // Content from InLines (if applicable)
	InputLineNo int    // Original line number from InLines (1-based)
	PatchLine   string // Content from DiffLines (if applicable, without operator)
	PatchLineNo int    // Original line number from DiffLines (1-based)
}

type EditCosts struct {
	InLines   []string
	DiffLines []string
	NI, ND    int
	costTable [][]int
	opTable   [][]EditOp
	visited   [][]bool

	// A helper to Checks if two strings are "almost" equal
	// we will use this for fuzzy similarity instead of absolute equality
	AlmostEqual func(s1, s2 string) bool
}

func (e *EditCosts) Init() *EditCosts {
	// only keep non empty diff lines
	e.DiffLines = fn.Filter(e.DiffLines, func(s string) bool {
		return len(s) > 0 && !strings.HasPrefix(s, "---") && !strings.HasPrefix(s, "+++")
	})
	e.NI = len(e.InLines)
	e.ND = len(e.DiffLines)

	e.costTable = make([][]int, e.NI+1)
	e.opTable = make([][]EditOp, e.NI+1)
	e.visited = make([][]bool, e.NI+1)
	for i := range e.NI + 1 {
		e.costTable[i] = make([]int, e.ND+1)
		e.opTable[i] = make([]EditOp, e.ND+1)
		e.visited[i] = make([]bool, e.ND+1)
	}

	// Base case: both sequences exhausted
	e.costTable[e.NI][e.ND] = 0
	e.opTable[e.NI][e.ND] = NOOP // Or a specific "END" op
	e.visited[e.NI][e.ND] = true

	// Case: DiffLines exhausted, InLines remain
	// These are input lines that the patch didn't touch further.
	for i := e.NI - 1; i >= 0; i-- {
		e.costTable[i][e.ND] = 0 + e.costTable[i+1][e.ND] // Cost 0 to keep an input line
		e.opTable[i][e.ND] = KEPT_INPUT_UNAFFECTED
		e.visited[i][e.ND] = true
	}

	// Case: InLines exhausted, DiffLines remain
	// These are patch operations that have no corresponding input lines.
	for d := e.ND - 1; d >= 0; d-- {
		costForThisDiffLine := 1 // Default cost for a remaining patch line (e.g. to skip it)
		op := NOOP               // Placeholder, should be more specific

		switch e.DiffLines[d][0] {
		case '+':
			// This is an add operation from the patch; it can still be "applied"
			costForThisDiffLine = 1 // Cost of an insertion
			op = ADDED_FROM_PATCH
		case '-':
			// A delete op with no input line to delete from; skip it
			costForThisDiffLine = 1      // Cost of skipping a patch delete op
			op = DELETE_SKIPPED_PATCH_OP // Or a more general SKIP_PATCH_OP
		case ' ':
			// A context op with no input line to match; skip it
			costForThisDiffLine = 1       // Cost of skipping a patch context op
			op = CONTEXT_SKIPPED_PATCH_OP // Or a more general SKIP_PATCH_OP
		default:
			// Invalid patch line marker
			costForThisDiffLine = 1 // Cost to skip invalid line
			op = NOOP               // Or SKIP_INVALID_PATCH_OP
		}
		e.costTable[e.NI][d] = costForThisDiffLine + e.costTable[e.NI][d+1]
		e.opTable[e.NI][d] = op
		e.visited[e.NI][d] = true
	}

	if e.AlmostEqual == nil {
		// default AlmostEqual to a space trimmed equality
		e.AlmostEqual = func(s1, s2 string) bool { return strings.TrimSpace(s1) == strings.TrimSpace(s2) }
		e.AlmostEqual = func(s1, s2 string) bool { return s1 == s2 }
	}
	return e
}

func (e *EditCosts) Dist() int {
	return e.Init().dist(0, 0)
}

func (e *EditCosts) dist(i, d int) int {
	if e.visited[i][d] {
		return e.costTable[i][d]
	}
	e.visited[i][d] = true

	currentBestOp := NOOP

	// Option 1: Keep InLines[i] and advance i (patch doesn't act on this InLines[i])
	// This is always a possibility, representing InLines[i] as an "original" line
	// that current DiffLines[d] doesn't consume.
	// Cost: Cost of keeping the input line + recurse.
	// This acts as a baseline or a way to skip over an input line if the patch line is better matched later.
	// Path A: Consume InLines[i] as KEPT_INPUT_UNAFFECTED
	costA := costKeepInputLine + e.dist(i+1, d)
	currentBestCost := costA
	currentBestOp = KEPT_INPUT_UNAFFECTED
	// if currentBestCost == -1 || costA < currentBestCost { }

	inline := e.InLines[i]
	diffline := e.DiffLines[d]
	dlContent := diffline[1:] // Content of diffline without the operator

	if diffline[0] == ' ' { // Patch context line
		// Path B: Try to match InLines[i] with this context line DiffLines[d]
		if e.AlmostEqual(dlContent, inline) {
			costB := costMatchContext + e.dist(i+1, d+1)
			if costB <= currentBestCost {
				currentBestCost = costB
				currentBestOp = CONTEXT_MATCHED
			}
		} else {
			// Context line doesn't match InLines[i].
			// Option B.1: Skip this patch context line and try to match InLines[i] with DiffLines[d+1].
			// Cost: Cost of skipping a patch context line + recurse.
			costB1 := costSkipPatchContext + e.dist(i, d+1)
			if costB1 < currentBestCost {
				currentBestCost = costB1
				currentBestOp = CONTEXT_SKIPPED_PATCH_OP // Implies InLines[i] is still available for next DiffLine
			}
			// Note: Path A (KEPT_INPUT_UNAFFECTED for InLines[i]) already covers the case where
			// InLines[i] is kept and we move to DiffLines[d] for the *next* InLine.
		}
	} else if diffline[0] == '+' { // Patch add line
		// Path C: Add DiffLines[d] from the patch. InLines[i] is not consumed by this operation.
		// Cost: Cost of adding a patch line + recurse.
		costC := costAddPatchLine + e.dist(i, d+1)
		if costC <= currentBestCost {
			currentBestCost = costC
			currentBestOp = ADDED_FROM_PATCH
		}
	} else if diffline[0] == '-' { // Patch delete line
		// Path D: Try to apply this delete operation to InLines[i]
		if e.AlmostEqual(dlContent, inline) {
			// Lines match for deletion.
			// Cost: Cost of a matched delete + recurse.
			costD1 := costMatchDelete + e.dist(i+1, d+1)
			// Prefer DELETED_INPUT_MATCHED_PATCH if cost is <= current (which might now be cost_skip_patch_del)
			if costD1 <= currentBestCost {
				currentBestCost = costD1
				currentBestOp = DELETED_INPUT_MATCHED_PATCH
			}
		} else if false {
			// Lines do NOT match, but patch wants to delete something.
			// Option D.2: Force delete InLines[i] anyway (fuzzy delete).
			// Cost: Cost of a fuzzy/unmatched delete + recurse.
			costD2 := costFuzzyDelete + e.dist(i+1, d+1)
			// Prefer DELETED_INPUT_UNMATCHED_PATCH if cost is <= current
			if costD2 <= currentBestCost {
				currentBestCost = costD2
				currentBestOp = DELETED_INPUT_UNMATCHED_PATCH
			}
		}

		if false {
			// Option E: Skip this patch delete operation. InLines[i] is not consumed by *this* delete op.
			// Try to match InLines[i] with DiffLines[d+1] or consider InLines[i] as KEPT_INPUT_UNAFFECTED.
			// Cost: Cost of skipping a patch delete operation + recurse.
			costE := costSkipPatchDelete + e.dist(i, d+1)
			if costE < currentBestCost {
				currentBestCost = costE
				currentBestOp = DELETE_SKIPPED_PATCH_OP // Implies InLines[i] is still available
			}
		}
	} else { // Invalid patch line marker
		// Path F: Skip this invalid patch line. InLines[i] is not consumed.
		// Cost: Cost of skipping an invalid patch line + recurse.
		costF := costSkipInvalidPatchLine + e.dist(i, d+1)
		if costF < currentBestCost {
			currentBestCost = costF
			currentBestOp = NOOP // Or a more specific SKIP_INVALID_PATCH_OP
		}
	}

	e.costTable[i][d] = currentBestCost
	e.opTable[i][d] = currentBestOp
	return e.costTable[i][d]
}

func (e *EditCosts) GetAlignment() []AlignedLine {
	alignment := make([]AlignedLine, 0)
	i, d := 0, 0

	for i < e.NI || d < e.ND {
		if i >= e.NI && d >= e.ND { // Both exhausted, should be handled by loop condition
			break
		}

		op := e.opTable[i][d] // Get pre-computed op from the table

		// Handle boundary conditions where one index is already at the end
		if i >= e.NI { // InLines exhausted, only DiffLines remain
			op = e.opTable[e.NI][d] // Use boundary condition op
		} else if d >= e.ND { // DiffLines exhausted, only InLines remain
			op = e.opTable[i][e.ND] // Use boundary condition op
		}
		// else op = e.opTable[i][d] // This was already done

		var currentAlignedLine AlignedLine
		currentAlignedLine.Type = op

		switch op {
		case KEPT_INPUT_UNAFFECTED:
			currentAlignedLine.InputLine = e.InLines[i]
			currentAlignedLine.InputLineNo = i + 1
			i++
		case CONTEXT_MATCHED:
			currentAlignedLine.InputLine = e.InLines[i]
			currentAlignedLine.InputLineNo = i + 1
			currentAlignedLine.PatchLine = e.DiffLines[d][1:]
			currentAlignedLine.PatchLineNo = d + 1
			i++
			d++
		case CONTEXT_SKIPPED_PATCH_OP: // DiffLines[d] (context) was skipped
			currentAlignedLine.PatchLine = e.DiffLines[d][1:] // Show it as skipped
			currentAlignedLine.PatchLineNo = d + 1
			// InLines[i] is NOT consumed by this op, it will be processed by the next iteration
			// with the same 'i' but 'd+1'.
			// However, our DP state assumes InLines[i] is available for the *next* DiffLine,
			// so if we record this skip, we should probably show InLines[i] first as KEPT
			// if opTable[i][d+1] indicates that. This reconstruction gets tricky.

			// Simpler: if this op means only 'd' advances, just mark the patch line.
			// The 'i' will be handled by subsequent ops.
			currentAlignedLine.InputLine = ""      // Or indicate it's about the patch line
			currentAlignedLine.InputLineNo = i + 1 // Still relevant to input line i
			d++
		case ADDED_FROM_PATCH:
			currentAlignedLine.PatchLine = e.DiffLines[d][1:]
			currentAlignedLine.PatchLineNo = d + 1
			currentAlignedLine.InputLineNo = i // Associated with being before/at input line i
			d++
		case DELETED_INPUT_MATCHED_PATCH, DELETED_INPUT_UNMATCHED_PATCH:
			currentAlignedLine.InputLine = e.InLines[i]
			currentAlignedLine.InputLineNo = i + 1
			currentAlignedLine.PatchLine = e.DiffLines[d][1:] // The delete instruction
			currentAlignedLine.PatchLineNo = d + 1
			i++
			d++
		case DELETE_SKIPPED_PATCH_OP: // DiffLines[d] (delete) was skipped
			currentAlignedLine.PatchLine = e.DiffLines[d][1:] // Show it as skipped
			currentAlignedLine.PatchLineNo = d + 1
			currentAlignedLine.InputLineNo = i + 1
			d++
		case NOOP: // Could be an invalid patch line skipped, or end of alignment
			if d < e.ND { // If it's skipping an invalid patch line
				currentAlignedLine.PatchLine = e.DiffLines[d] // Show the raw invalid line
				currentAlignedLine.PatchLineNo = d + 1
				d++
			} else if i < e.NI { // Should be KEPT_INPUT_UNAFFECTED if only InLines left
				// This case might indicate an issue or end of useful alignment.
				// For safety, break or handle as KEPT_INPUT_UNAFFECTED if i is valid.
				currentAlignedLine.InputLine = e.InLines[i]
				currentAlignedLine.InputLineNo = i + 1
				currentAlignedLine.Type = KEPT_INPUT_UNAFFECTED // Override
				i++
			} else { // Both i and d are at NI, ND
				// This is the actual end, opTable[NI][ND] should be NOOP
				if i == e.NI && d == e.ND && e.opTable[i][d] == NOOP {
					// Normal termination
				} else {
					// Log error or handle unexpected state
				}
				goto endLoop // Exit loop
			}

		default:
			// Should not happen if opTable is filled correctly
			// Log error or panic
			panic("unknown op code during reconstruction")
		}
		alignment = append(alignment, currentAlignedLine)
	}
endLoop:
	return alignment
}

// ApplyPatchToInput takes the original input lines and patch diff lines,
// performs the alignment, and returns the lines that would result from
// applying the patch operations.
func (e *EditCosts) ApplyPatchToInput() []string {
	// Ensure Dist() has been called to populate tables, or call it if not.
	// Calling Init() again is safe if it's idempotent regarding already filled tables,
	// or we can rely on the user calling Dist() or GetAlignment() first.
	// For simplicity, let's assume Init() and Dist() have been effectively called
	// by the time GetAlignment() is valid. Typically, you'd call e.Dist() once.
	// If GetAlignment is called standalone, it should ensure tables are ready.
	// Let's make GetAlignment ensure readiness.

	// (No, GetAlignment should assume tables are ready from a prior Dist() call)
	// The caller of ApplyPatchToInput should do:
	// ec := newEditCosts(inLines, diffLines)
	// ec.Dist() // This calculates costs and fills opTable
	// outputLines := ec.ApplyPatchToInput()

	alignment := e.GetAlignment()
	outputLines := make([]string, 0)

	for _, aligned := range alignment {
		switch aligned.Type {
		case KEPT_INPUT_UNAFFECTED:
			outputLines = append(outputLines, aligned.InputLine)
		case CONTEXT_MATCHED:
			// Context lines are essentially kept input lines that were confirmed by the patch
			outputLines = append(outputLines, aligned.InputLine)
		// case CONTEXT_SKIPPED_PATCH_OP:
		// If a context patch line was skipped, the corresponding input line
		// was likely KEPT_INPUT_UNAFFECTED in a different step or will be.
		// This operation itself doesn't add to output, it's informational.
		// The input line it might have referred to is handled by other ops.
		// No direct output line from this specific AlignedLine type.
		case ADDED_FROM_PATCH:
			outputLines = append(outputLines, aligned.PatchLine)
		case DELETED_INPUT_MATCHED_PATCH:
			// Line is deleted, so nothing is added to outputLines
			break
		case DELETED_INPUT_UNMATCHED_PATCH:
			// Line is considered deleted (even if it didn't match patch perfectly),
			// so nothing is added to outputLines
			break
		// case DELETE_SKIPPED_PATCH_OP:
		// If a delete patch line was skipped, the corresponding input line
		// was likely KEPT_INPUT_UNAFFECTED or handled by other ops.
		// No direct output line from this specific AlignedLine type.
		case NOOP:
			// Generally, NOOP at the end or for skipped invalid lines doesn't contribute to output.
			// If it represented an invalid patch line that was skipped, we don't add it.
			break
		default:
			// Handle any other op codes if they are meant to produce output.
			// For now, we assume unlisted ops don't directly contribute to the final output lines.
			// Or, if an unknown op appears, it might be an error.
			// For robustness, you might want to decide if unknown ops should panic
			// or be ignored.
		}
	}

	return outputLines
}
