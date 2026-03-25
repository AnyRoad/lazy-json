package session

import (
	"errors"
	"testing"

	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/perftest"
	"github.com/anyroad/lazy-json/internal/source"
)

func loadBenchmarkDocument(b *testing.B, fixture string) *document.Document {
	b.Helper()

	doc, path, err := perftest.LoadDocument(fixture)
	if err != nil {
		if errors.Is(err, perftest.ErrFixtureUnavailable) {
			b.Skipf("skipping benchmark; fixture unavailable: %s", path)
		}
		b.Fatalf("LoadDocument(%q) error = %v", fixture, err)
	}
	return doc
}

func firstContainerChild(node *document.Node) *document.Node {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case document.KindObject:
		for _, entry := range node.Object {
			if entry.Value != nil && entry.Value.IsContainer() {
				return entry.Value
			}
		}
	case document.KindArray:
		for _, child := range node.Array {
			if child != nil && child.IsContainer() {
				return child
			}
		}
	}
	return nil
}

func BenchmarkSessionRefresh(b *testing.B) {
	b.Run("huge-array/root", func(b *testing.B) {
		doc := loadBenchmarkDocument(b, perftest.FixtureHugeArray)
		s := New(doc, source.Input{Kind: source.KindFile, Path: "hugeArray.json"}, "")

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Refresh(doc)
		}
	})

	b.Run("huge-array/first-batch-open", func(b *testing.B) {
		doc := loadBenchmarkDocument(b, perftest.FixtureHugeArray)
		s := New(doc, source.Input{Kind: source.KindFile, Path: "hugeArray.json"}, "")
		s.ExpandedBatches[BatchRowID(doc.Root.ID, 0)] = true
		s.Refresh(doc)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Refresh(doc)
		}
	})

	b.Run("huge-json/root", func(b *testing.B) {
		doc := loadBenchmarkDocument(b, perftest.FixtureHugeJSON)
		s := New(doc, source.Input{Kind: source.KindFile, Path: "hugeJson.json"}, "")

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Refresh(doc)
		}
	})

	b.Run("huge-json/first-container-open", func(b *testing.B) {
		doc := loadBenchmarkDocument(b, perftest.FixtureHugeJSON)
		s := New(doc, source.Input{Kind: source.KindFile, Path: "hugeJson.json"}, "")
		if child := firstContainerChild(doc.Root); child != nil {
			s.Expanded[child.ID] = true
		}
		s.Refresh(doc)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Refresh(doc)
		}
	})
}

func BenchmarkSessionSearchHits(b *testing.B) {
	b.Run("huge-array/address", func(b *testing.B) {
		doc := loadBenchmarkDocument(b, perftest.FixtureHugeArray)
		s := New(doc, source.Input{Kind: source.KindFile, Path: "hugeArray.json"}, "")
		s.Search.Query = "address"

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.UpdateSearchHits(doc)
		}
	})

	b.Run("huge-json/trainStation", func(b *testing.B) {
		doc := loadBenchmarkDocument(b, perftest.FixtureHugeJSON)
		s := New(doc, source.Input{Kind: source.KindFile, Path: "hugeJson.json"}, "")
		s.Search.Query = "trainstation"

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.UpdateSearchHits(doc)
		}
	})
}
