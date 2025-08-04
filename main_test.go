package shardedflight

import (
	"errors"
	"fmt"
	"golang.org/x/sync/singleflight"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// // // // // // // //

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		shards  uint32
		wantErr bool
	}{
		{"valid-1", 1, false},
		{"valid-2", 2, false},
		{"valid-4", 4, false},
		{"valid-16", 16, false},
		{"valid-256", 256, false},
		{"invalid-0", 0, true},
		{"invalid-3", 3, true},
		{"invalid-5", 5, true},
		{"invalid-6", 6, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := ConfObj{Shards: tt.shards}
			obj, err := New(conf)

			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error for Shards=%d, got nil", tt.shards)
				}
			} else {
				if err != nil {
					t.Errorf("New() unexpected error for Shards=%d: %v", tt.shards, err)
				}
				if obj == nil {
					t.Errorf("New() returned nil obj for valid configuration")
				} else {
					if len(obj.shards) != int(tt.shards) {
						t.Errorf("New() created incorrect number of shards: got %d, expected %d", len(obj.shards), tt.shards)
					}
					if obj.mask != uint64(tt.shards-1) {
						t.Errorf("New() set incorrect mask: got %d, expected %d", obj.mask, tt.shards-1)
					}
				}
			}
		})
	}
}

func TestNew_CustomFunctions(t *testing.T) {
	customBuilder := func(parts ...string) string {
		return "custom"
	}
	customHash := func(s string) uint64 {
		return 42
	}

	conf := ConfObj{
		Shards:   4,
		BuildKey: customBuilder,
		Hash:     customHash,
	}
	obj, err := New(conf)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	if reflect.ValueOf(obj.conf.BuildKey).Pointer() != reflect.ValueOf(customBuilder).Pointer() {
		t.Errorf("New() did not preserve custom BuildKey function")
	}
	if reflect.ValueOf(obj.conf.Hash).Pointer() != reflect.ValueOf(customHash).Pointer() {
		t.Errorf("New() did not preserve custom Hash function")
	}
}

func TestDefaultBuilder(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{"empty", []string{}, ""},
		{"single", []string{"test"}, "test"},
		{"multiple", []string{"hello", "world"}, "helloworld"},
		{"three-parts", []string{"a", "b", "c"}, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultBuilder(tt.parts...)
			if result != tt.expected {
				t.Errorf("defaultBuilder() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestNewDo(t *testing.T) {
	obj, err := New(ConfObj{Shards: 4})
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	key := "test-key"
	expectedResult := "result"

	result, err, shared := obj.Do(func() (any, error) {
		return expectedResult, nil
	}, key)

	if err != nil {
		t.Errorf("Do() unexpected error: %v", err)
	}
	if result != expectedResult {
		t.Errorf("Do() = %v, expected %v", result, expectedResult)
	}
	if shared {
		t.Errorf("Do() first call should not be shared")
	}

	if count := obj.InFlight(); count != 0 {
		t.Errorf("InFlight() = %d after completed Do(), expected 0", count)
	}

	expectedErr := errors.New("test error")
	_, err, _ = obj.Do(func() (any, error) {
		return nil, expectedErr
	}, key)

	if err != expectedErr {
		t.Errorf("Do() error = %v, expected %v", err, expectedErr)
	}
}

// TestDo_Concurrent checks that concurrent calls with the same key
// are deduplicated correctly. For this we use only one shard
// to ensure that all requests go to it.
func TestDo_Concurrent(t *testing.T) {
	obj, err := New(ConfObj{Shards: 1}) // Використовуємо тільки один шард
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	const numCalls = 100
	const numGoroutines = 10
	key := "concurrent-key"

	callCount := 0
	var countMutex sync.Mutex

	var results = make([]any, numCalls)
	var errs = make([]error, numCalls)
	var shareds = make([]bool, numCalls)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < numCalls/numGoroutines; i++ {
				index := goroutineID*(numCalls/numGoroutines) + i

				before := obj.InFlight()

				results[index], errs[index], shareds[index] = obj.Do(func() (any, error) {
					countMutex.Lock()
					callCount++
					currentCount := callCount
					countMutex.Unlock()

					time.Sleep(10 * time.Millisecond)

					return fmt.Sprintf("result-%d", currentCount), nil
				}, key)

				after := obj.InFlight()
				if before < 0 || after < 0 {
					t.Errorf("InFlight() returned negative value: before=%d, after=%d", before, after)
				}
			}
		}(g)
	}

	wg.Wait()

	if callCount >= 50 {
		t.Errorf("Function was called %d times, expected 1 time due to deduplication", callCount)
	}

	sharedCount := 0
	for _, shared := range shareds {
		if shared {
			sharedCount++
		}
	}

	if sharedCount != numCalls {
		t.Errorf("Got %d shared calls, expected %d", sharedCount, numCalls)
	}

	if count := obj.InFlight(); count != 0 {
		t.Errorf("InFlight() = %d after all calls completed, expected 0", count)
	}
}

func TestDo_DifferentKeys(t *testing.T) {
	obj, err := New(ConfObj{Shards: 4})
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	counts := make(map[string]int)
	var countMutex sync.Mutex

	makeFunc := func(key string) func() (any, error) {
		return func() (any, error) {
			countMutex.Lock()
			counts[key]++
			countMutex.Unlock()
			time.Sleep(1 * time.Millisecond)
			return key, nil
		}
	}

	var wg sync.WaitGroup
	keys := []string{"key1", "key2", "key3", "key4"}

	for _, key := range keys {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(k string) {
				defer wg.Done()
				obj.Do(makeFunc(k), k)
			}(key)
		}
	}

	wg.Wait()

	for _, key := range keys {
		if counts[key] != 1 {
			t.Errorf("Function for key %q was called %d times, expected 1 time", key, counts[key])
		}
	}
}

// TestDo_ShardDeduplication checks that requests with the same keys
// are deduplicated correctly if they fall into the same shard.
func TestDo_ShardDeduplication(t *testing.T) {
	// Створюємо настроювану хеш-функцію, яка повертає різні хеші
	// для різних ключів, але один і той самий хеш для ключів з однаковим префіксом
	customHash := func(s string) uint64 {
		if strings.HasPrefix(s, "shard-0-") {
			return 0
		} else if strings.HasPrefix(s, "shard-1-") {
			return 1
		} else if strings.HasPrefix(s, "shard-2-") {
			return 2
		} else {
			return 3
		}
	}

	obj, err := New(ConfObj{
		Shards: 4,
		Hash:   customHash,
	})
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	// Створюємо кілька горутин, які викликають Do з однаковими ключами
	// для кожного шарду. Функція повинна бути викликана рівно один раз для кожного ключа.

	const numShards = 4
	const numKeysPerShard = 3
	const numCallsPerKey = 10

	callCounts := make(map[string]int)
	var countMutex sync.Mutex

	var wg sync.WaitGroup

	for shard := 0; shard < numShards; shard++ {
		for keyIdx := 0; keyIdx < numKeysPerShard; keyIdx++ {
			key := fmt.Sprintf("shard-%d-key-%d", shard, keyIdx)

			wg.Add(numCallsPerKey)
			for call := 0; call < numCallsPerKey; call++ {
				go func(k string) {
					defer wg.Done()

					obj.Do(func() (any, error) {
						countMutex.Lock()
						callCounts[k]++
						countMutex.Unlock()

						time.Sleep(1 * time.Millisecond)

						return k, nil
					}, k)
				}(key)
			}
		}
	}

	wg.Wait()

	// Перевіряємо, що функція була викликана рівно один раз для кожного ключа

	for shard := 0; shard < numShards; shard++ {
		for keyIdx := 0; keyIdx < numKeysPerShard; keyIdx++ {
			key := fmt.Sprintf("shard-%d-key-%d", shard, keyIdx)

			count, ok := callCounts[key]
			if !ok {
				t.Errorf("Key %q is missing in callCounts", key)
			} else if count != 1 {
				t.Errorf("Function for key %q was called %d times, expected 1 time", key, count)
			}
		}
	}
}

func TestNewDoChan(t *testing.T) {
	obj, err := New(ConfObj{Shards: 4})
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	key := "test-key"
	expectedResult := "result"

	resultChan := obj.DoChan(func() (any, error) {
		return expectedResult, nil
	}, key)

	result := <-resultChan

	if result.Err != nil {
		t.Errorf("DoChan() unexpected error: %v", result.Err)
	}
	if result.Val != expectedResult {
		t.Errorf("DoChan() = %v, expected %v", result.Val, expectedResult)
	}
	if result.Shared {
		t.Errorf("DoChan() first call should not be shared")
	}

	if count := obj.InFlight(); count != 0 {
		t.Errorf("InFlight() = %d after completed DoChan(), expected 0", count)
	}

	expectedErr := errors.New("test error")
	resultChan = obj.DoChan(func() (any, error) {
		return nil, expectedErr
	}, key)

	result = <-resultChan

	if result.Err != expectedErr {
		t.Errorf("DoChan() error = %v, expected %v", result.Err, expectedErr)
	}
}

// TestDoChan_Concurrent checks that concurrent calls to DoChan with the same key
// are deduplicated correctly. For this we use only one shard
// to ensure that all requests go to it.
func TestDoChan_Concurrent(t *testing.T) {
	obj, err := New(ConfObj{Shards: 1}) // Використовуємо тільки один шард
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	const numCalls = 100
	const numGoroutines = 10
	key := "concurrent-key"

	callCount := 0
	var countMutex sync.Mutex

	var results = make([]any, numCalls)
	var errs = make([]error, numCalls)
	var shareds = make([]bool, numCalls)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < numCalls/numGoroutines; i++ {
				index := goroutineID*(numCalls/numGoroutines) + i

				before := obj.InFlight()

				resultChan := obj.DoChan(func() (any, error) {
					countMutex.Lock()
					callCount++
					currentCount := callCount
					countMutex.Unlock()

					time.Sleep(1 * time.Millisecond)

					return fmt.Sprintf("result-%d", currentCount), nil
				}, key)

				result := <-resultChan
				results[index] = result.Val
				errs[index] = result.Err
				shareds[index] = result.Shared

				after := obj.InFlight()
				if before < 0 || after < 0 {
					t.Errorf("InFlight() returned negative value: before=%d, after=%d", before, after)
				}
			}
		}(g)
	}

	wg.Wait()

	if callCount >= 50 {
		t.Errorf("Function was called %d times, expected 1 time due to deduplication", callCount)
	}

	sharedCount := 0
	for _, shared := range shareds {
		if shared {
			sharedCount++
		}
	}

	if sharedCount != numCalls {
		t.Errorf("Got %d shared calls, expected %d", sharedCount, numCalls)
	}

	if count := obj.InFlight(); count != 0 {
		t.Errorf("InFlight() = %d after all calls completed, expected 0", count)
	}
}

func TestInFlight(t *testing.T) {
	obj, err := New(ConfObj{Shards: 4})
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	if count := obj.InFlight(); count != 0 {
		t.Errorf("Initial value of InFlight() = %d, expected 0", count)
	}

	control := make(chan struct{})
	done := make(chan struct{})

	go func() {
		obj.Do(func() (any, error) {
			<-control
			return "done", nil
		}, "test-key")
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)

	if count := obj.InFlight(); count != 1 {
		t.Errorf("InFlight() during execution = %d, expected 1", count)
	}

	close(control)
	<-done

	if count := obj.InFlight(); count != 0 {
		t.Errorf("InFlight() after execution = %d, expected 0", count)
	}
}

func TestInFlight_MultipleCalls(t *testing.T) {
	obj, err := New(ConfObj{Shards: 4})
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	const numCalls = 10
	controlChans := make([]chan struct{}, numCalls)
	doneChans := make([]chan struct{}, numCalls)

	for i := 0; i < numCalls; i++ {
		controlChans[i] = make(chan struct{})
		doneChans[i] = make(chan struct{})

		go func(idx int) {
			obj.Do(func() (any, error) {
				<-controlChans[idx]
				return idx, nil
			}, fmt.Sprintf("key-%d", idx))
			close(doneChans[idx])
		}(i)
	}

	time.Sleep(10 * time.Millisecond)

	if count := obj.InFlight(); count != numCalls {
		t.Errorf("InFlight() during execution = %d, expected %d", count, numCalls)
	}

	for i := 0; i < numCalls; i++ {
		close(controlChans[i])
		<-doneChans[i]

		expected := numCalls - (i + 1)
		if count := obj.InFlight(); count != int64(expected) {
			t.Errorf("InFlight() after %d completions = %d, expected %d", i+1, count, expected)
		}
	}
}

// // // //

func BenchmarkDo(b *testing.B) {
	obj, err := New(ConfObj{Shards: 16})
	if err != nil {
		b.Fatalf("New() unexpected error: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%100)
		obj.Do(func() (any, error) {
			return "result", nil
		}, key)
	}
}

func BenchmarkDo_Parallel(b *testing.B) {
	obj, err := New(ConfObj{Shards: 16})
	if err != nil {
		b.Fatalf("New() unexpected error: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", counter%100)
			counter++
			obj.Do(func() (any, error) {
				return "result", nil
			}, key)
		}
	})
}

func BenchmarkDoChan(b *testing.B) {
	obj, err := New(ConfObj{Shards: 16})
	if err != nil {
		b.Fatalf("New() unexpected error: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%100)
		ch := obj.DoChan(func() (any, error) {
			return "result", nil
		}, key)
		<-ch
	}
}

func BenchmarkDoChan_Parallel(b *testing.B) {
	obj, err := New(ConfObj{Shards: 16})
	if err != nil {
		b.Fatalf("New() unexpected error: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", counter%100)
			counter++
			ch := obj.DoChan(func() (any, error) {
				return "result", nil
			}, key)
			<-ch
		}
	})
}

func BenchmarkDefaultBuilder(b *testing.B) {
	b.Run("Empty", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			defaultBuilder()
		}
	})

	b.Run("Single", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			defaultBuilder("test")
		}
	})

	b.Run("Double", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			defaultBuilder("hello", "world")
		}
	})

	b.Run("Multiple", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			defaultBuilder("a", "b", "c", "d", "e")
		}
	})
}

func BenchmarkComparison_SingleFlight(b *testing.B) {
	var group singleflight.Group

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", counter%100)
			counter++
			group.Do(key, func() (any, error) {
				return "result", nil
			})
		}
	})
}

func BenchmarkComparison_ShardedFlight(b *testing.B) {
	obj, _ := New(ConfObj{Shards: 16})

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", counter%100)
			counter++
			obj.Do(func() (any, error) {
				return "result", nil
			}, key)
		}
	})
}

func BenchmarkShardCount(b *testing.B) {
	shardCounts := []uint32{1, 2, 4, 8, 16, 32, 64, 128, 256}

	for _, shards := range shardCounts {
		b.Run(fmt.Sprintf("Shards-%d", shards), func(b *testing.B) {
			obj, _ := New(ConfObj{Shards: shards})

			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				counter := 0
				for pb.Next() {
					key := fmt.Sprintf("key-%d", counter%1000)
					counter++
					obj.Do(func() (any, error) {
						return "result", nil
					}, key)
				}
			})
		})
	}
}

func BenchmarkCustomFunctions(b *testing.B) {
	b.Run("Default", func(b *testing.B) {
		obj, _ := New(ConfObj{Shards: 16})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i%100)
			obj.Do(func() (any, error) {
				return "result", nil
			}, key)
		}
	})

	b.Run("CustomBuildKey", func(b *testing.B) {
		customBuilder := func(parts ...string) string {
			if len(parts) == 0 {
				return ""
			}
			if len(parts) == 1 {
				return parts[0]
			}
			var sb strings.Builder
			for _, p := range parts {
				sb.WriteString(p)
			}
			return sb.String()
		}

		obj, _ := New(ConfObj{
			Shards:   16,
			BuildKey: customBuilder,
		})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i%100)
			obj.Do(func() (any, error) {
				return "result", nil
			}, key)
		}
	})

	b.Run("CustomHash", func(b *testing.B) {
		customHash := func(s string) uint64 {
			h := 0
			for i := 0; i < len(s); i++ {
				h = 31*h + int(s[i])
			}
			return uint64(h)
		}

		obj, _ := New(ConfObj{
			Shards: 16,
			Hash:   customHash,
		})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i%100)
			obj.Do(func() (any, error) {
				return "result", nil
			}, key)
		}
	})

	b.Run("CustomBoth", func(b *testing.B) {
		customBuilder := func(parts ...string) string {
			if len(parts) == 0 {
				return ""
			}
			if len(parts) == 1 {
				return parts[0]
			}
			var sb strings.Builder
			for _, p := range parts {
				sb.WriteString(p)
			}
			return sb.String()
		}

		customHash := func(s string) uint64 {
			h := 0
			for i := 0; i < len(s); i++ {
				h = 31*h + int(s[i])
			}
			return uint64(h)
		}

		obj, _ := New(ConfObj{
			Shards:   16,
			BuildKey: customBuilder,
			Hash:     customHash,
		})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i%100)
			obj.Do(func() (any, error) {
				return "result", nil
			}, key)
		}
	})
}
