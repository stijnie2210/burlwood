package repo

import (
	"reflect"
	"testing"
)

func TestParseTags(t *testing.T) {
	out := "v0.2.0\taaaa\tcccc\nv0.1.0\tbbbb\t\n"
	want := []TagInfo{
		{Name: "v0.2.0", Hash: "cccc"},
		{Name: "v0.1.0", Hash: "bbbb"},
	}
	if got := parseTags(out); !reflect.DeepEqual(got, want) {
		t.Fatalf("parseTags = %#v, want %#v", got, want)
	}
}

func TestParseTagsEmpty(t *testing.T) {
	if got := parseTags(""); len(got) != 0 {
		t.Fatalf("parseTags(\"\") = %#v, want empty", got)
	}
}
