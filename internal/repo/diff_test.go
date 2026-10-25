package repo

import (
	"testing"
)

// --- parseHunkHeader ---

func TestParseHunkHeader_Basic(t *testing.T) {
	old, new_ := parseHunkHeader("@@ -1,4 +1,5 @@")
	if old != 1 {
		t.Errorf("oldStart = %d, want 1", old)
	}
	if new_ != 1 {
		t.Errorf("newStart = %d, want 1", new_)
	}
}

func TestParseHunkHeader_WithContext(t *testing.T) {
	old, new_ := parseHunkHeader("@@ -10,7 +10,9 @@ func Foo() {")
	if old != 10 {
		t.Errorf("oldStart = %d, want 10", old)
	}
	if new_ != 10 {
		t.Errorf("newStart = %d, want 10", new_)
	}
}

func TestParseHunkHeader_NewFile(t *testing.T) {
	old, new_ := parseHunkHeader("@@ -0,0 +1,3 @@")
	if old != 0 {
		t.Errorf("oldStart = %d, want 0", old)
	}
	if new_ != 1 {
		t.Errorf("newStart = %d, want 1", new_)
	}
}

// --- parseGitStatus ---

func TestParseGitStatus_Empty(t *testing.T) {
	files := parseGitStatus("")
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestParseGitStatus_ModifiedUnstaged(t *testing.T) {
	files := parseGitStatus(" M foo.go\n")
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f := files[0]
	if f.Path != "foo.go" {
		t.Errorf("path = %q, want %q", f.Path, "foo.go")
	}
	if f.Staged != " " {
		t.Errorf("staged = %q, want ' '", f.Staged)
	}
	if f.Unstaged != "M" {
		t.Errorf("unstaged = %q, want 'M'", f.Unstaged)
	}
}

func TestParseGitStatus_StagedModified(t *testing.T) {
	files := parseGitStatus("M  bar.go\n")
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f := files[0]
	if f.Staged != "M" {
		t.Errorf("staged = %q, want 'M'", f.Staged)
	}
	if f.Unstaged != " " {
		t.Errorf("unstaged = %q, want ' '", f.Unstaged)
	}
}

func TestParseGitStatus_Untracked(t *testing.T) {
	files := parseGitStatus("?? new.go\n")
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f := files[0]
	if f.Path != "new.go" {
		t.Errorf("path = %q, want %q", f.Path, "new.go")
	}
	if f.Staged != "?" || f.Unstaged != "?" {
		t.Errorf("staged=%q unstaged=%q, want '?' '?'", f.Staged, f.Unstaged)
	}
}

func TestParseGitStatus_Deleted(t *testing.T) {
	files := parseGitStatus(" D gone.go\n")
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Unstaged != "D" {
		t.Errorf("unstaged = %q, want 'D'", files[0].Unstaged)
	}
}

func TestParseGitStatus_Renamed(t *testing.T) {
	files := parseGitStatus("R  old.go -> new.go\n")
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f := files[0]
	if f.Path != "new.go" {
		t.Errorf("path = %q, want %q", f.Path, "new.go")
	}
	if f.OldPath != "old.go" {
		t.Errorf("oldPath = %q, want %q", f.OldPath, "old.go")
	}
}

func TestParseGitStatus_MultipleFiles(t *testing.T) {
	input := " M foo.go\n?? bar.go\nM  baz.go\n"
	files := parseGitStatus(input)
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d: %v", len(files), files)
	}
	if files[0].Path != "foo.go" || files[1].Path != "bar.go" || files[2].Path != "baz.go" {
		t.Errorf("paths = %v/%v/%v", files[0].Path, files[1].Path, files[2].Path)
	}
}

func TestParseGitStatus_ConflictBothModified(t *testing.T) {
	files := parseGitStatus("UU conflict.go\n")
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f := files[0]
	if f.Staged != "U" || f.Unstaged != "U" {
		t.Errorf("staged=%q unstaged=%q, want 'U' 'U'", f.Staged, f.Unstaged)
	}
}

func TestParseGitStatus_IgnoresShortLines(t *testing.T) {
	files := parseGitStatus("\n \n")
	if len(files) != 0 {
		t.Errorf("expected 0 files from short/empty lines, got %d", len(files))
	}
}

// --- parseUnifiedDiff ---

func TestParseUnifiedDiff_Empty(t *testing.T) {
	diffs := parseUnifiedDiff("")
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs, got %d", len(diffs))
	}
}

func TestParseUnifiedDiff_SimpleDiff(t *testing.T) {
	input := `diff --git a/foo.go b/foo.go
index abc..def 100644
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,4 @@
 context
-deleted line
+added line
 more context
+new line
`
	diffs := parseUnifiedDiff(input)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 FileDiff, got %d", len(diffs))
	}
	d := diffs[0]
	if d.OldPath != "foo.go" {
		t.Errorf("oldPath = %q, want %q", d.OldPath, "foo.go")
	}
	if d.NewPath != "foo.go" {
		t.Errorf("newPath = %q, want %q", d.NewPath, "foo.go")
	}
	if d.IsBinary {
		t.Error("should not be binary")
	}
	if len(d.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(d.Hunks))
	}

	lines := d.Hunks[0].Lines
	types := make([]string, len(lines))
	for i, l := range lines {
		types[i] = l.Type
	}
	want := []string{"context", "del", "add", "context", "add"}
	if len(types) != len(want) {
		t.Fatalf("line types = %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Errorf("line %d type = %q, want %q", i, types[i], want[i])
		}
	}
}

func TestParseUnifiedDiff_NewFile(t *testing.T) {
	input := `diff --git a/newfile.go b/newfile.go
new file mode 100644
--- /dev/null
+++ b/newfile.go
@@ -0,0 +1,2 @@
+line1
+line2
`
	diffs := parseUnifiedDiff(input)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 FileDiff, got %d", len(diffs))
	}
	d := diffs[0]
	if d.OldPath != "" {
		t.Errorf("oldPath = %q, want empty (new file)", d.OldPath)
	}
	if d.NewPath != "newfile.go" {
		t.Errorf("newPath = %q, want %q", d.NewPath, "newfile.go")
	}
	if len(d.Hunks[0].Lines) != 2 {
		t.Errorf("expected 2 add lines, got %d", len(d.Hunks[0].Lines))
	}
	for _, l := range d.Hunks[0].Lines {
		if l.Type != "add" {
			t.Errorf("line type = %q, want add", l.Type)
		}
	}
}

func TestParseUnifiedDiff_DeletedFile(t *testing.T) {
	input := `diff --git a/old.go b/old.go
deleted file mode 100644
--- a/old.go
+++ /dev/null
@@ -1,2 +0,0 @@
-line1
-line2
`
	diffs := parseUnifiedDiff(input)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 FileDiff, got %d", len(diffs))
	}
	d := diffs[0]
	if d.OldPath != "old.go" {
		t.Errorf("oldPath = %q, want %q", d.OldPath, "old.go")
	}
	if d.NewPath != "" {
		t.Errorf("newPath = %q, want empty (deleted file)", d.NewPath)
	}
	for _, l := range d.Hunks[0].Lines {
		if l.Type != "del" {
			t.Errorf("line type = %q, want del", l.Type)
		}
	}
}

func TestParseUnifiedDiff_BinaryFile(t *testing.T) {
	input := `diff --git a/image.png b/image.png
index abc..def 100644
Binary files a/image.png and b/image.png differ
`
	diffs := parseUnifiedDiff(input)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 FileDiff, got %d", len(diffs))
	}
	if !diffs[0].IsBinary {
		t.Error("expected IsBinary = true")
	}
}

func TestParseUnifiedDiff_BinaryPaths(t *testing.T) {
	input := `diff --git a/fonts/word mark.woff b/fonts/word mark.woff
new file mode 100644
index 0000000..abc
Binary files /dev/null and b/fonts/word mark.woff differ
diff --git a/old.png b/old.png
deleted file mode 100644
index abc..0000000
Binary files a/old.png and /dev/null differ
diff --git a/icon.png b/icon.png
index abc..def 100644
Binary files a/icon.png and b/icon.png differ
`
	diffs := parseUnifiedDiff(input)
	if len(diffs) != 3 {
		t.Fatalf("expected 3 FileDiffs, got %d", len(diffs))
	}
	want := [][2]string{{"", "fonts/word mark.woff"}, {"old.png", ""}, {"icon.png", "icon.png"}}
	for i, w := range want {
		if diffs[i].OldPath != w[0] || diffs[i].NewPath != w[1] {
			t.Errorf("diff %d paths = (%q, %q), want (%q, %q)", i, diffs[i].OldPath, diffs[i].NewPath, w[0], w[1])
		}
	}
}

func TestParseUnifiedDiff_MultipleFiles(t *testing.T) {
	input := `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,1 @@
-old
+new
diff --git a/b.go b/b.go
--- a/b.go
+++ b/b.go
@@ -1,1 +1,1 @@
-foo
+bar
`
	diffs := parseUnifiedDiff(input)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 FileDiffs, got %d", len(diffs))
	}
	if diffs[0].OldPath != "a.go" {
		t.Errorf("diffs[0].oldPath = %q, want a.go", diffs[0].OldPath)
	}
	if diffs[1].OldPath != "b.go" {
		t.Errorf("diffs[1].oldPath = %q, want b.go", diffs[1].OldPath)
	}
}

func TestParseUnifiedDiff_MultipleHunks(t *testing.T) {
	input := `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,3 @@
 context
-del1
+add1
@@ -10,3 +10,3 @@
 context
-del2
+add2
`
	diffs := parseUnifiedDiff(input)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 FileDiff, got %d", len(diffs))
	}
	if len(diffs[0].Hunks) != 2 {
		t.Errorf("expected 2 hunks, got %d", len(diffs[0].Hunks))
	}
}

func TestParseUnifiedDiff_NoNewlineAtEOF(t *testing.T) {
	// The "\ No newline at end of file" line should be silently ignored
	input := `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,1 @@
-old
\ No newline at end of file
+new
\ No newline at end of file
`
	diffs := parseUnifiedDiff(input)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 FileDiff, got %d", len(diffs))
	}
	lines := diffs[0].Hunks[0].Lines
	// Should only have del and add, not the "\ No newline" lines
	for _, l := range lines {
		if l.Type != "del" && l.Type != "add" {
			t.Errorf("unexpected line type %q, content %q", l.Type, l.Content)
		}
	}
}

func TestParseUnifiedDiff_LineNumbers(t *testing.T) {
	input := `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -5,3 +5,4 @@
 context
+added
 context2
-deleted
`
	diffs := parseUnifiedDiff(input)
	lines := diffs[0].Hunks[0].Lines

	// context at old=5, new=5
	if lines[0].OldLine != 5 || lines[0].NewLine != 5 {
		t.Errorf("context line: old=%d new=%d, want 5,5", lines[0].OldLine, lines[0].NewLine)
	}
	// added: old=0 (no old line), new=6
	if lines[1].OldLine != 0 || lines[1].NewLine != 6 {
		t.Errorf("add line: old=%d new=%d, want 0,6", lines[1].OldLine, lines[1].NewLine)
	}
	// context2: old=6, new=7
	if lines[2].OldLine != 6 || lines[2].NewLine != 7 {
		t.Errorf("context2 line: old=%d new=%d, want 6,7", lines[2].OldLine, lines[2].NewLine)
	}
	// deleted: old=7, new=0
	if lines[3].OldLine != 7 || lines[3].NewLine != 0 {
		t.Errorf("del line: old=%d new=%d, want 7,0", lines[3].OldLine, lines[3].NewLine)
	}
}
