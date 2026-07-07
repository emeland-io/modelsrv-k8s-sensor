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
