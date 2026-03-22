package document

import (
	"strings"
	"testing"
)

func TestResolvePathExactMatches(t *testing.T) {
	doc, err := Parse([]byte(`{"items":[{"title":"alpha"},{"title":"beta"}],"two words":[{"quote\"x":true}]}`))
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		path         string
		wantID       NodeID
		wantMatched  string
		wantExact    bool
		wantAncestor []NodeID
	}{
		{
			path:         "$",
			wantID:       doc.Root.ID,
			wantMatched:  "$",
			wantExact:    true,
			wantAncestor: nil,
		},
		{
			path:         "$.items[1].title",
			wantID:       doc.Root.Object[0].Value.Array[1].Object[0].Value.ID,
			wantMatched:  "$.items[1].title",
			wantExact:    true,
			wantAncestor: []NodeID{doc.Root.ID, doc.Root.Object[0].Value.ID, doc.Root.Object[0].Value.Array[1].ID},
		},
		{
			path:         `$["two words"][0]["quote\"x"]`,
			wantID:       doc.Root.Object[1].Value.Array[0].Object[0].Value.ID,
			wantMatched:  `$["two words"][0]["quote\"x"]`,
			wantExact:    true,
			wantAncestor: []NodeID{doc.Root.ID, doc.Root.Object[1].Value.ID, doc.Root.Object[1].Value.Array[0].ID},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.path, func(t *testing.T) {
			resolution, err := doc.ResolvePath(testCase.path)
			if err != nil {
				t.Fatalf("ResolvePath(%q) error = %v", testCase.path, err)
			}
			if got, want := resolution.NodeID, testCase.wantID; got != want {
				t.Fatalf("NodeID = %d, want %d", got, want)
			}
			if got, want := resolution.MatchedPath, testCase.wantMatched; got != want {
				t.Fatalf("MatchedPath = %q, want %q", got, want)
			}
			if got, want := resolution.Exact, testCase.wantExact; got != want {
				t.Fatalf("Exact = %v, want %v", got, want)
			}
			if got, want := resolution.Ancestors, testCase.wantAncestor; len(got) != len(want) {
				t.Fatalf("Ancestors = %v, want %v", got, want)
			} else {
				for index := range want {
					if got[index] != want[index] {
						t.Fatalf("Ancestors[%d] = %d, want %d", index, got[index], want[index])
					}
				}
			}
		})
	}
}

func TestResolvePathFallsBackToNearestExistingAncestor(t *testing.T) {
	doc, err := Parse([]byte(`{"items":[{"title":"alpha"},{"title":"beta"}],"meta":{"count":2}}`))
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name         string
		path         string
		wantID       NodeID
		wantMatched  string
		wantAncestor []NodeID
	}{
		{
			name:         "missing array index",
			path:         "$.items[9].title",
			wantID:       doc.Root.Object[0].Value.ID,
			wantMatched:  "$.items",
			wantAncestor: []NodeID{doc.Root.ID},
		},
		{
			name:         "wrong container kind",
			path:         "$.items.title",
			wantID:       doc.Root.Object[0].Value.ID,
			wantMatched:  "$.items",
			wantAncestor: []NodeID{doc.Root.ID},
		},
		{
			name:         "root only",
			path:         "$.missing.branch",
			wantID:       doc.Root.ID,
			wantMatched:  "$",
			wantAncestor: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resolution, err := doc.ResolvePath(testCase.path)
			if err != nil {
				t.Fatalf("ResolvePath(%q) error = %v", testCase.path, err)
			}
			if resolution.Exact {
				t.Fatal("Exact = true, want false")
			}
			if got, want := resolution.NodeID, testCase.wantID; got != want {
				t.Fatalf("NodeID = %d, want %d", got, want)
			}
			if got, want := resolution.MatchedPath, testCase.wantMatched; got != want {
				t.Fatalf("MatchedPath = %q, want %q", got, want)
			}
			if got, want := resolution.Ancestors, testCase.wantAncestor; len(got) != len(want) {
				t.Fatalf("Ancestors = %v, want %v", got, want)
			} else {
				for index := range want {
					if got[index] != want[index] {
						t.Fatalf("Ancestors[%d] = %d, want %d", index, got[index], want[index])
					}
				}
			}
		})
	}
}

func TestResolvePathRejectsMalformedSyntax(t *testing.T) {
	doc, err := Parse([]byte(`{"items":[1,2]}`))
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		path string
		want string
	}{
		{path: "", want: "path cannot be empty"},
		{path: "items[0]", want: "path must start with $"},
		{path: "$.", want: "expected identifier after ."},
		{path: "$.items[", want: "expected array index or quoted key after ["},
		{path: "$.items[0", want: "expected ] after array index"},
		{path: `$["items]`, want: "unterminated quoted key"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.path, func(t *testing.T) {
			_, err := doc.ResolvePath(testCase.path)
			if err == nil {
				t.Fatalf("ResolvePath(%q) error = nil", testCase.path)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("ResolvePath(%q) error = %q, want substring %q", testCase.path, err.Error(), testCase.want)
			}
		})
	}
}
