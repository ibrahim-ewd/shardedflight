package shardedflight

import (
	"reflect"
	"testing"
)

// // // // // // // //

func TestName(t *testing.T) {
	t.Log(defaultHash("test"))
	t.Log(18007334074686647077 == 18007334074686647077)
}

func TestNew_DefaultFunctions(t *testing.T) {
	conf := ConfObj{Shards: 4}
	obj, err := New(conf)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	if reflect.ValueOf(obj.conf.BuildKey).Pointer() != reflect.ValueOf(defaultBuilder).Pointer() {
		t.Errorf("New() did not set default BuildKey function")
	}
	if reflect.ValueOf(obj.conf.Hash).Pointer() != reflect.ValueOf(defaultHash).Pointer() {
		t.Errorf("New() did not set default Hash function")
	}
}

func TestDefaultHash(t *testing.T) {
	h1 := defaultHash("test")
	h2 := defaultHash("test")
	if h1 != h2 {
		t.Errorf("defaultHash() should return the same hash for identical strings: got %d and %d", h1, h2)
	}

	h3 := defaultHash("hello")
	if h1 == h3 {
		t.Errorf("defaultHash() returned the same hash for different strings: %d", h1)
	}
}

// // // //

func BenchmarkDefaultHash(b *testing.B) {
	b.Run("ShortString", func(b *testing.B) {
		str := "test"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			defaultHash(str)
		}
	})

	b.Run("LongString", func(b *testing.B) {
		str := "this is a long string that we use to test the performance of the hash function with more characters"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			defaultHash(str)
		}
	})
}
