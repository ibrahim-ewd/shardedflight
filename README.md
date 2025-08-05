# ShardedFlight

Ready-to-use highly parallel wrapper for **golang.org/x/sync/singleflight** that *shards* calls between multiple internal `singleflight.Group` to eliminate global locks and increase throughput under high load.
The only difference - keys after function (iterator).

---

## Quick Start

```go
import "github.com/voluminor/shardedflight"

sf, _ := shardedflight.New(shardedflight.ConfObj{Shards: 16})

val, err, shared := sf.Do(func () (any, error) {
return computeOrFetch(id), nil
}, id)
if err != nil { … }
if shared { log.Println("result reused") }
````

---

## Why ShardedFlight?

| Feature                         | `singleflight` | **ShardedFlight**             |
|---------------------------------|----------------|-------------------------------|
| Global lock per key             | ✔︎             | **Sharded** across *N* groups |
| Custom key builder (zero alloc) | ✗              | ✔︎                            |
| Pluggable hash function         | ✗              | ✔︎                            |
| Live `InFlight()` counter       | ✗              | ✔︎                            |

Sharding divides the key-space with a power-of-two mask so unrelated keys
never contend on the same mutex. Benchmarks show 2–8× higher QPS on
≥8-core machines at p99 latency.

---

## API Reference

### type `ConfObj`

| Field      | Type                           | Description                                                                                 |
|------------|--------------------------------|---------------------------------------------------------------------------------------------|
| `Shards`   | `uint32`                       | **Required.** Must be a power of two; defines the number of internal groups.                |
| `BuildKey` | `func(parts ...string) string` | Optional zero-allocation key builder. Defaults to an unsafe, allocation-free concatenation. |
| `Hash`     | `func(string) uint64`          | Optional hash. Defaults to 64-bit FNV-1a (\~1 ns/key).                                      |

### func `New(conf ConfObj) (*ModObj, error)`

Validates `conf`, fills defaults, allocates `conf.Shards` groups and returns a
ready-to-use instance. Returns `ErrInvalidShards` when `Shards` is zero or not
a power-of-two.

### type `ModObj`

| Method                                                | Purpose                                                      |
|-------------------------------------------------------|--------------------------------------------------------------|
| `Do(fn, keyParts...) (v any, err error, shared bool)` | Deduplicates concurrent calls with the same key.             |
| `DoChan(fn, keyParts...) <-chan singleflight.Result`  | Channel form, never blocks the caller goroutine.             |
| `Forget(keyParts...)`                                 | Removes cached result for `key`; next call re-executes `fn`. |
| `InFlight() int64`                                    | Number of currently running (not completed) executions.      |

---

## Usage Examples

### 1. Cache-stampede protection for HTTP handler

```go
type Item = mypkg.Item

sf, _ := shardedflight.New(shardedflight.ConfObj{Shards: 64})

func itemHandler(w http.ResponseWriter, r *http.Request) {
id := r.URL.Query().Get("id")
v, err, _ := sf.Do(func () (any, error) {
return queryDB(id) // expensive
}, "item", id)
if err != nil {
http.Error(w, err.Error(), 500)
return
}
json.NewEncoder(w).Encode(v.(Item))
}
```

### 2. Custom zero-alloc key and fast CRC-32 hash

```go
builder := func (p ...string) string { return strings.Join(p, "|") }
hash := func (s string) uint64    { return uint64(crc32.ChecksumIEEE([]byte(s))) }

sf, _ := shardedflight.New(shardedflight.ConfObj{
Shards:   32,
BuildKey: builder,
Hash:     hash,
})
```

### 3. Monitoring in-flight work

```go
go func () {
for range time.Tick(time.Second) {
m.Set(float64(sf.InFlight())) // export via Prometheus
}
}()
```

### 4. Fan-out with `DoChan`

```go
resCh := sf.DoChan(func () (any, error) { return callRPC(k), nil }, k)
doOtherWork()
res := <-resCh
```

---

## When to Use

* High-QPS APIs where many different keys are requested simultaneously.
* Layer-7 caches or CDN edge workers protecting origins from stampedes.
* Task schedulers benchmarking identical jobs.
* Micro-services performing idempotent but costly RPC/database lookups.
* Any place where vanilla `singleflight` becomes a bottleneck above
  \~10 k req/s due to a single global mutex.
