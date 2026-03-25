package tui

import (
	"errors"
	"testing"

	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/perftest"
	"github.com/anyroad/lazy-json/internal/session"
	"github.com/anyroad/lazy-json/internal/source"
)

func loadBenchmarkModelDocument(b *testing.B, fixture string) *document.Document {
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

func firstRootContainerChild(node *document.Node) *document.Node {
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

func newBenchmarkModel(doc *document.Document, path string) *Model {
	model := NewModel(doc, source.Input{Kind: source.KindFile, Path: path}, ModelOptions{})
	model.Width = 120
	model.Height = 40
	return model
}

func BenchmarkModelView(b *testing.B) {
	b.Run("huge-array/default", func(b *testing.B) {
		doc := loadBenchmarkModelDocument(b, perftest.FixtureHugeArray)
		model := newBenchmarkModel(doc, "hugeArray.json")

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = model.View()
		}
	})

	b.Run("huge-array/expanded-first-batch", func(b *testing.B) {
		doc := loadBenchmarkModelDocument(b, perftest.FixtureHugeArray)
		model := newBenchmarkModel(doc, "hugeArray.json")
		model.Session.ExpandedBatches[session.BatchRowID(doc.Root.ID, 0)] = true
		model.Session.Refresh(doc)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = model.View()
		}
	})

	b.Run("huge-array/active-search", func(b *testing.B) {
		doc := loadBenchmarkModelDocument(b, perftest.FixtureHugeArray)
		model := newBenchmarkModel(doc, "hugeArray.json")
		model.Session.Search.Query = "address"
		model.Session.UpdateSearchHits(doc)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = model.View()
		}
	})

	b.Run("huge-json/default", func(b *testing.B) {
		doc := loadBenchmarkModelDocument(b, perftest.FixtureHugeJSON)
		model := newBenchmarkModel(doc, "hugeJson.json")

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = model.View()
		}
	})

	b.Run("huge-json/expanded-first-container", func(b *testing.B) {
		doc := loadBenchmarkModelDocument(b, perftest.FixtureHugeJSON)
		model := newBenchmarkModel(doc, "hugeJson.json")
		if child := firstRootContainerChild(doc.Root); child != nil {
			model.Session.Expanded[child.ID] = true
		}
		model.Session.Refresh(doc)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = model.View()
		}
	})

	b.Run("huge-json/active-search", func(b *testing.B) {
		doc := loadBenchmarkModelDocument(b, perftest.FixtureHugeJSON)
		model := newBenchmarkModel(doc, "hugeJson.json")
		model.Session.Search.Query = "trainstation"
		model.Session.UpdateSearchHits(doc)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = model.View()
		}
	})
}
