package document

import (
	"errors"
	"testing"
)

func TestParsePreservesOrderAndNumbers(t *testing.T) {
	doc, err := Parse([]byte(`{"b":1,"a":2,"big":12345678901234567890}`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if doc.Root.Kind != KindObject {
		t.Fatalf("Root kind = %q", doc.Root.Kind)
	}
	keys := []string{doc.Root.Object[0].Key, doc.Root.Object[1].Key, doc.Root.Object[2].Key}
	want := []string{"b", "a", "big"}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys[%d] = %q, want %q", i, keys[i], want[i])
		}
	}
	if got := doc.Root.Object[2].Value.Number; got != "12345678901234567890" {
		t.Fatalf("Number = %q", got)
	}
}

func TestMarshalIndent(t *testing.T) {
	doc, err := Parse([]byte(`{"name":"Ada","list":[1,true,null]}`))
	if err != nil {
		t.Fatal(err)
	}
	got, err := doc.MarshalIndent()
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"name\": \"Ada\",\n  \"list\": [\n    1,\n    true,\n    null\n  ]\n}\n"
	if string(got) != want {
		t.Fatalf("MarshalIndent() = %q, want %q", got, want)
	}
}

func TestMarshalIndentWithIndent(t *testing.T) {
	doc, err := Parse([]byte(`{"name":"Ada","list":[1,true]}`))
	if err != nil {
		t.Fatal(err)
	}

	got, err := doc.MarshalIndentWith("\t")
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n\t\"name\": \"Ada\",\n\t\"list\": [\n\t\t1,\n\t\ttrue\n\t]\n}\n"
	if string(got) != want {
		t.Fatalf("MarshalIndentWith(tab) = %q, want %q", got, want)
	}

	node, err := MarshalIndentNodeWithIndent(doc.Root, "   ")
	if err != nil {
		t.Fatal(err)
	}
	wantNode := "{\n   \"name\": \"Ada\",\n   \"list\": [\n      1,\n      true\n   ]\n}\n"
	if string(node) != wantNode {
		t.Fatalf("MarshalIndentNodeWithIndent(3 spaces) = %q, want %q", node, wantNode)
	}
}

func TestEditOperations(t *testing.T) {
	doc, err := Parse([]byte(`{"name":"Ada","list":[1]}`))
	if err != nil {
		t.Fatal(err)
	}
	nameID := doc.Root.Object[0].Value.ID
	listID := doc.Root.Object[1].Value.ID

	if err := doc.RenameKey(nameID, "full_name"); err != nil {
		t.Fatalf("RenameKey() error = %v", err)
	}
	replacement, err := ParseNode([]byte(`"Grace"`))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Replace(nameID, replacement); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	if got := doc.Root.Object[0].Value.String; got != "Grace" {
		t.Fatalf("replaced string = %q", got)
	}
	added, err := ParseNode([]byte(`2`))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.AddArrayItem(listID, added); err != nil {
		t.Fatalf("AddArrayItem() error = %v", err)
	}
	if len(doc.Root.Object[1].Value.Array) != 2 {
		t.Fatalf("array length = %d", len(doc.Root.Object[1].Value.Array))
	}
	flag, err := ParseNode([]byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.AddObjectEntry(doc.Root.ID, "active", flag); err != nil {
		t.Fatalf("AddObjectEntry() error = %v", err)
	}
	if len(doc.Root.Object) != 3 {
		t.Fatalf("object length = %d", len(doc.Root.Object))
	}
	if err := doc.Delete(flag.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(doc.Root.Object) != 2 {
		t.Fatalf("object length after delete = %d", len(doc.Root.Object))
	}
}

func TestDeleteRootFails(t *testing.T) {
	doc, err := Parse([]byte(`{"name":"Ada"}`))
	if err != nil {
		t.Fatal(err)
	}
	err = doc.Delete(doc.Root.ID)
	if !errors.Is(err, ErrDeleteRoot) {
		t.Fatalf("Delete() error = %v, want %v", err, ErrDeleteRoot)
	}
}
