package graph

import (
	"testing"
)

func node(hash string, parents ...string) CommitNode {
	return CommitNode{Hash: hash, Subject: "msg", Author: "a", Date: "d", ParentHashes: parents}
}

func TestBuildGraph_Empty(t *testing.T) {
	rows := BuildGraph(nil, nil)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestBuildGraph_SingleRootCommit(t *testing.T) {
	rows := BuildGraph([]CommitNode{node("aaaaaaa")}, nil)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.Column != 0 {
		t.Errorf("root commit: column = %d, want 0", r.Column)
	}
	if r.Color == "" {
		t.Error("root commit: color should be assigned")
	}
	if len(r.Edges) != 0 {
		t.Errorf("root commit: edges = %v, want none", r.Edges)
	}
	if len(r.EdgesIn) != 0 {
		t.Errorf("root commit: edgesIn = %v, want none (first row)", r.EdgesIn)
	}
	if r.ShortHash != "aaaaaaa" {
		t.Errorf("shortHash = %q, want %q", r.ShortHash, "aaaaaaa")
	}
}

// Linear chain A→B→C (log order: A, B, C where A is newest)
func TestBuildGraph_LinearChain(t *testing.T) {
	commits := []CommitNode{
		node("aaaaaaa", "bbbbbbb"),
		node("bbbbbbb", "ccccccc"),
		node("ccccccc"),
	}
	rows := BuildGraph(commits, nil)

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	for i, r := range rows {
		if r.Column != 0 {
			t.Errorf("row %d: column = %d, want 0", i, r.Column)
		}
	}

	// Each row except the last should have exactly one outgoing edge going straight down (col 0→0)
	for i := range 2 {
		if len(rows[i].Edges) != 1 {
			t.Errorf("row %d: len(edges) = %d, want 1", i, len(rows[i].Edges))
			continue
		}
		e := rows[i].Edges[0]
		if e.FromCol != 0 || e.ToCol != 0 {
			t.Errorf("row %d edge: {%d→%d}, want {0→0}", i, e.FromCol, e.ToCol)
		}
	}

	// EdgesIn of row i should equal Edges of row i-1
	for i := 1; i < len(rows); i++ {
		if len(rows[i].EdgesIn) != len(rows[i-1].Edges) {
			t.Errorf("row %d: edgesIn len = %d, want %d", i, len(rows[i].EdgesIn), len(rows[i-1].Edges))
		}
	}
}

// Two branches diverging from the same parent:
//
//   - A (col 0, parent C)
//   - B (col 1, parent C — opens new lane)
//   - C (col 0, root)
func TestBuildGraph_DivergingBranches(t *testing.T) {
	commits := []CommitNode{
		node("aaaaaaa", "ccccccc"),
		node("bbbbbbb", "ccccccc"),
		node("ccccccc"),
	}
	rows := BuildGraph(commits, nil)

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	if rows[0].Column != 0 {
		t.Errorf("A column = %d, want 0", rows[0].Column)
	}
	if rows[1].Column != 1 {
		t.Errorf("B column = %d, want 1", rows[1].Column)
	}
	if rows[2].Column != 0 {
		t.Errorf("C column = %d, want 0", rows[2].Column)
	}

	// B converges toward C's lane (col 0), so one of B's edges should go {1→0}
	bEdgesTo0 := false
	for _, e := range rows[1].Edges {
		if e.FromCol == 1 && e.ToCol == 0 {
			bEdgesTo0 = true
		}
	}
	if !bEdgesTo0 {
		t.Errorf("B should have edge {1→0} converging to C's lane, got %v", rows[1].Edges)
	}
}

// Merge commit with two parents:
//
//   - C (merge, parents=[A,B])
//   - A
//   - B
func TestBuildGraph_MergeCommit(t *testing.T) {
	commits := []CommitNode{
		node("ccccccc", "aaaaaaa", "bbbbbbb"),
		node("aaaaaaa"),
		node("bbbbbbb"),
	}
	rows := BuildGraph(commits, nil)

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// C should open a second lane for the second parent B
	// Edges from C: one straight {0→0} for A, one diverging {0→?} for B
	cRow := rows[0]
	if len(cRow.Edges) < 2 {
		t.Errorf("merge commit: want at least 2 edges (one per parent), got %d", len(cRow.Edges))
	}
}

// Lane reuse: after a branch closes its lane, the slot should be reused.
//
//   - D (col 0, parent C)
//   - B (col 1, no parent — branch tip)
//   - C (col 0, parent A)
//   - A (col 0, root)
func TestBuildGraph_LaneReuse(t *testing.T) {
	commits := []CommitNode{
		node("ddddddd", "ccccccc"),
		node("bbbbbbb"),
		node("ccccccc", "aaaaaaa"),
		node("aaaaaaa"),
	}
	rows := BuildGraph(commits, nil)

	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}

	// B takes col 1 (D is in col 0 waiting for C)
	if rows[1].Column != 1 {
		t.Errorf("B column = %d, want 1", rows[1].Column)
	}

	// After B closes (no parents), lane 1 is freed.
	// C (col 0) should not require a new lane beyond index 1.
	if rows[2].MaxColumn > 1 {
		t.Errorf("C maxColumn = %d; lane reuse expected, want ≤1", rows[2].MaxColumn)
	}
}

func TestBuildGraph_Labels(t *testing.T) {
	labels := map[string][]Label{
		"aaaaaaa": {{Name: "main", Type: "head"}},
		"bbbbbbb": {{Name: "v1.0", Type: "tag"}},
	}
	commits := []CommitNode{
		node("aaaaaaa", "bbbbbbb"),
		node("bbbbbbb"),
	}
	rows := BuildGraph(commits, labels)

	if len(rows[0].Labels) != 1 || rows[0].Labels[0].Name != "main" {
		t.Errorf("row 0 labels = %v, want [{main head}]", rows[0].Labels)
	}
	if len(rows[1].Labels) != 1 || rows[1].Labels[0].Name != "v1.0" {
		t.Errorf("row 1 labels = %v, want [{v1.0 tag}]", rows[1].Labels)
	}
}

func TestBuildGraph_NoLabelReturnsEmptySlice(t *testing.T) {
	rows := BuildGraph([]CommitNode{node("aaaaaaa")}, nil)
	if rows[0].Labels == nil {
		t.Error("labels should be empty slice, not nil")
	}
}

func TestBuildGraph_EdgesInSecondPass(t *testing.T) {
	commits := []CommitNode{
		node("aaaaaaa", "bbbbbbb"),
		node("bbbbbbb", "ccccccc"),
		node("ccccccc"),
	}
	rows := BuildGraph(commits, nil)

	if len(rows[0].EdgesIn) != 0 {
		t.Errorf("first row EdgesIn should be empty, got %v", rows[0].EdgesIn)
	}
	for i := 1; i < len(rows); i++ {
		prev := rows[i-1].Edges
		cur := rows[i].EdgesIn
		if len(cur) != len(prev) {
			t.Errorf("row %d: EdgesIn len %d != prev Edges len %d", i, len(cur), len(prev))
			continue
		}
		for j := range prev {
			if cur[j] != prev[j] {
				t.Errorf("row %d edge %d: EdgesIn=%v, want %v", i, j, cur[j], prev[j])
			}
		}
	}
}

func TestBuildGraph_MaxColumn(t *testing.T) {
	// Two parallel branches — max column should reach 1
	commits := []CommitNode{
		node("aaaaaaa", "ccccccc"),
		node("bbbbbbb", "ccccccc"),
		node("ccccccc"),
	}
	rows := BuildGraph(commits, nil)

	// B (row 1) has edges crossing col 0 and col 1
	if rows[1].MaxColumn < 1 {
		t.Errorf("B maxColumn = %d, want ≥1 (two active lanes)", rows[1].MaxColumn)
	}
}
