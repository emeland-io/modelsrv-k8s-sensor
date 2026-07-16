package controller

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNameIndexPutGetDelete(t *testing.T) {
	idx := NewNameIndex()
	id := uuid.New()

	idx.Put(KindSystem, "default/my-sys", id)
	assert.Equal(t, id, idx.Get(KindSystem, "default/my-sys"))

	got := idx.Delete(KindSystem, "default/my-sys")
	assert.Equal(t, id, got)
	assert.Equal(t, uuid.Nil, idx.Get(KindSystem, "default/my-sys"))
}

func TestNameIndexUnknownKind(t *testing.T) {
	idx := NewNameIndex()
	assert.Equal(t, uuid.Nil, idx.Get(KindAPI, "missing"))
	assert.Equal(t, uuid.Nil, idx.Delete(KindAPI, "missing"))
}

func TestNameIndexConcurrent(t *testing.T) {
	idx := NewNameIndex()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := uuid.New()
			name := "ns/res"
			idx.Put(KindComponent, name, id)
			_ = idx.Get(KindComponent, name)
			_ = idx.Delete(KindComponent, name)
		}()
	}
	wg.Wait()
}

func TestHelmOwnerSetGetDelete(t *testing.T) {
	idx := NewNameIndex()
	siID := uuid.New()

	idx.SetHelmOwner(KindComponentInstance, "ns/deploy", siID)
	assert.Equal(t, siID, idx.GetHelmOwner(KindComponentInstance, "ns/deploy"))
	assert.Equal(t, uuid.Nil, idx.GetHelmOwner(KindComponentInstance, "ns/other"))
	assert.Equal(t, uuid.Nil, idx.GetHelmOwner(KindAPIInstance, "ns/deploy"))

	idx.DeleteHelmOwner(KindComponentInstance, "ns/deploy")
	assert.Equal(t, uuid.Nil, idx.GetHelmOwner(KindComponentInstance, "ns/deploy"))
}

func TestHelmOwnerDeleteBySystemInstance(t *testing.T) {
	idx := NewNameIndex()
	si1 := uuid.New()
	si2 := uuid.New()

	idx.SetHelmOwner(KindComponentInstance, "ns/a", si1)
	idx.SetHelmOwner(KindComponentInstance, "ns/b", si1)
	idx.SetHelmOwner(KindAPIInstance, "ns/svc", si1)
	idx.SetHelmOwner(KindComponentInstance, "ns/c", si2)

	idx.DeleteHelmOwnersBySystemInstance(si1)

	assert.Equal(t, uuid.Nil, idx.GetHelmOwner(KindComponentInstance, "ns/a"))
	assert.Equal(t, uuid.Nil, idx.GetHelmOwner(KindComponentInstance, "ns/b"))
	assert.Equal(t, uuid.Nil, idx.GetHelmOwner(KindAPIInstance, "ns/svc"))
	// si2 entries should be untouched
	assert.Equal(t, si2, idx.GetHelmOwner(KindComponentInstance, "ns/c"))
}
