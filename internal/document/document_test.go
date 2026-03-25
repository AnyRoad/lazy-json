package document

import (
	"errors"
	"fmt"
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

func TestCloneNodePreservesIDsWithoutAliasing(t *testing.T) {
	doc, err := Parse([]byte(`{"items":[{"name":"Ada"}]}`))
	if err != nil {
		t.Fatal(err)
	}

	clone := CloneNode(doc.Root)
	if clone == doc.Root {
		t.Fatal("CloneNode() returned original pointer")
	}
	if got, want := clone.ID, doc.Root.ID; got != want {
		t.Fatalf("clone root ID = %d, want %d", got, want)
	}
	if got, want := clone.Object[0].Value.ID, doc.Root.Object[0].Value.ID; got != want {
		t.Fatalf("clone child ID = %d, want %d", got, want)
	}
	if got, want := clone.Object[0].Value.Array[0].Object[0].Value.ID, doc.Root.Object[0].Value.Array[0].Object[0].Value.ID; got != want {
		t.Fatalf("clone grandchild ID = %d, want %d", got, want)
	}

	clone.Object[0].Value.Array[0].Object[0].Value.String = "Grace"
	if got, want := doc.Root.Object[0].Value.Array[0].Object[0].Value.String, "Ada"; got != want {
		t.Fatalf("original string after clone mutation = %q, want %q", got, want)
	}
}

func TestRestorePreservesSubtreeIDs(t *testing.T) {
	doc, err := Parse([]byte(`{"profile":{"name":"Ada","city":"Seoul"}}`))
	if err != nil {
		t.Fatal(err)
	}

	profile := doc.Root.Object[0].Value
	nameID := profile.Object[0].Value.ID
	updated := CloneNode(profile)
	updated.Object[0].Value.String = "Grace"

	if err := doc.Restore(profile.ID, updated); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	restored := doc.Root.Object[0].Value
	if got, want := restored.Object[0].Value.String, "Grace"; got != want {
		t.Fatalf("restored string = %q, want %q", got, want)
	}
	if got, want := restored.Object[0].Value.ID, nameID; got != want {
		t.Fatalf("restored child ID = %d, want %d", got, want)
	}
}

func TestRestoreRejectsInvalidReplacement(t *testing.T) {
	doc, err := Parse([]byte(`{"profile":{"name":"Ada"}}`))
	if err != nil {
		t.Fatal(err)
	}

	profileID := doc.Root.Object[0].Value.ID

	t.Run("nil node", func(t *testing.T) {
		err := doc.Restore(profileID, nil)
		if got, want := err.Error(), "replacement node cannot be nil"; got != want {
			t.Fatalf("Restore() error = %q, want %q", got, want)
		}
	})

	t.Run("mismatched root id", func(t *testing.T) {
		err := doc.Restore(profileID, &Node{ID: profileID + 1, Kind: KindObject})
		want := fmt.Sprintf("replacement root id %d does not match target %d", profileID+1, profileID)
		if got := err.Error(); got != want {
			t.Fatalf("Restore() error = %q, want %q", got, want)
		}
	})
}

func TestInsertOperations(t *testing.T) {
	doc, err := Parse([]byte(`{"name":"Ada","list":[2]}`))
	if err != nil {
		t.Fatal(err)
	}

	rootID := doc.Root.ID
	listID := doc.Root.Object[1].Value.ID

	flag, err := ParseNode([]byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.InsertObjectEntryAt(rootID, 1, "active", flag); err != nil {
		t.Fatalf("InsertObjectEntryAt() error = %v", err)
	}

	if got, want := len(doc.Root.Object), 3; got != want {
		t.Fatalf("object length = %d, want %d", got, want)
	}
	if got, want := doc.Root.Object[1].Key, "active"; got != want {
		t.Fatalf("inserted object key = %q, want %q", got, want)
	}

	one, err := ParseNode([]byte(`1`))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.InsertArrayItemAt(listID, 0, one); err != nil {
		t.Fatalf("InsertArrayItemAt() error = %v", err)
	}

	if got, want := len(doc.Root.Object[2].Value.Array), 2; got != want {
		t.Fatalf("array length = %d, want %d", got, want)
	}
	if got, want := doc.Root.Object[2].Value.Array[0].Number, "1"; got != want {
		t.Fatalf("inserted array item = %q, want %q", got, want)
	}
	if got, want := doc.Root.Object[2].Value.Array[1].Number, "2"; got != want {
		t.Fatalf("shifted array item = %q, want %q", got, want)
	}
}

func TestInsertOperationsPreserveReservedIDs(t *testing.T) {
	doc, err := Parse([]byte(`{"list":[]}`))
	if err != nil {
		t.Fatal(err)
	}

	listID := doc.Root.Object[0].Value.ID
	preserved := &Node{ID: 99, Kind: KindNumber, Number: "7"}
	if err := doc.InsertArrayItemAt(listID, 0, preserved); err != nil {
		t.Fatalf("InsertArrayItemAt(preserved) error = %v", err)
	}

	appended, err := ParseNode([]byte(`8`))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.AddArrayItem(listID, appended); err != nil {
		t.Fatalf("AddArrayItem() error = %v", err)
	}

	if got, want := doc.Root.Object[0].Value.Array[0].ID, NodeID(99); got != want {
		t.Fatalf("preserved ID = %d, want %d", got, want)
	}
	if got := doc.Root.Object[0].Value.Array[1].ID; got <= 99 {
		t.Fatalf("newly assigned ID = %d, want > 99", got)
	}
}

func TestInsertOperationsErrorCases(t *testing.T) {
	doc, err := Parse([]byte(`{"name":"Ada","list":[]}`))
	if err != nil {
		t.Fatal(err)
	}

	rootID := doc.Root.ID
	listID := doc.Root.Object[1].Value.ID

	t.Run("object index out of range", func(t *testing.T) {
		err := doc.InsertObjectEntryAt(rootID, 3, "active", &Node{Kind: KindBool, Boolean: true})
		if got, want := err.Error(), "object insert index 3 out of range"; got != want {
			t.Fatalf("InsertObjectEntryAt() error = %q, want %q", got, want)
		}
	})

	t.Run("array index out of range", func(t *testing.T) {
		err := doc.InsertArrayItemAt(listID, 1, &Node{Kind: KindNumber, Number: "1"})
		if got, want := err.Error(), "array insert index 1 out of range"; got != want {
			t.Fatalf("InsertArrayItemAt() error = %q, want %q", got, want)
		}
	})

	t.Run("wrong parent kind", func(t *testing.T) {
		err := doc.InsertArrayItemAt(rootID, 0, &Node{Kind: KindNull})
		if got, want := err.Error(), "node 1 is not an array"; got != want {
			t.Fatalf("InsertArrayItemAt() error = %q, want %q", got, want)
		}
	})

	t.Run("nil node", func(t *testing.T) {
		err := doc.InsertObjectEntryAt(rootID, 1, "active", nil)
		if got, want := err.Error(), "insert node cannot be nil"; got != want {
			t.Fatalf("InsertObjectEntryAt() error = %q, want %q", got, want)
		}
	})
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
