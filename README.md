# shardedflight
ShardedFlight is a high-throughput, shard-aware drop-in replacement for golang.org/x/sync/singleflight. It distributes keys across N internal singleflight.Groups to eliminate global-lock contention, provides zero-allocation key building and pluggable hash functions, and exposes a live InFlight() counter for observability.
