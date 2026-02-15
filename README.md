https://github.com/ibrahim-ewd/shardedflight/releases

# ShardedFlight: High-Throughput Go Call Deduping with Internal Shards

[![Go Version](https://img.shields.io/badge/Go-1.20%2B-blue)](https://golang.org)
[![Releases](https://img.shields.io/badge/releases-download-blue?logo=github)](https://github.com/ibrahim-ewd/shardedflight/releases)

ShardedFlight is a ready-to-use, highly parallel wrapper around golang.org/x/sync/singleflight. It shards calls across multiple internal singleflight.Group instances to eliminate global locks and boost throughput under heavy load. The key difference is that keys are passed after the function, enabling flexible and expressive request patterns while preserving deduplication semantics.

This repository is a practical tool for building high-performance services in Go. It targets developers who want to reduce contention during bursty workloads, minimize lock contention, and scale duplicate suppression across many goroutines.

---

## Table of contents

- [Why ShardedFlight](#why-shardedflight)
- [What it is and what it isn’t](#what-it-is-and-what-it-isnt)
- [Key concepts and design decisions](#key-concepts-and-design-decisions)
- [Getting started](#getting-started)
  - [Installation](#installation)
  - [Quick start example](#quick-start-example)
- [API overview](#api-overview)
- [Deep dive into architecture](#deep-dive-into-architecture)
- [Usage patterns and examples](#usage-patterns-and-examples)
- [Tuning and best practices](#tuning-and-best-practices)
- [Performance and benchmarks](#performance-and-benchmarks)
- [Testing and reliability](#testing-and-reliability)
- [Extending and customizing](#extending-and-customizing)
- [Roadmap and future work](#roadmap-and-future-work)
- [Community and contribution](#community-and-contribution)
- [FAQ](#faq)
- [License](#license)
- [Releases](#releases)

---

## Why ShardedFlight

High-throughput apps face bursts of concurrent requests that perform identical or similar work. The classic approach uses a single global lock around a shared cache or a single deduplication pool. Under heavy load, this can become a bottleneck, causing increased latency and reduced throughput.

ShardedFlight hides that bottleneck by splitting the deduplication surface into multiple independent groups. Each shard handles a portion of the keys, allowing many blocking operations to proceed in parallel. This approach preserves the singleflight API semantics while spreading contention across multiple internal workers.

- Lock-free or low-lock design at scale
- Reduced contention during peak traffic
- Simple integration with existing code paths that rely on singleflight
- Flexible mapping of keys to shards, enabling nuanced routing strategies

The library emphasizes calm, robust performance. It is not meant to replace every use of singleflight but to offer a practical path when there is a need to scale deduplication and reduce global lock pressure in high-load scenarios.

---

## What it is and what it isn’t

What it is:
- A drop-in compatible wrapper that shards calls across several internal singleflight groups.
- A tool to increase throughput when many concurrent requests deduplicate work for identical keys.
- A practical design for Go services that face cache stampede-like patterns in distributed or multi-service environments.
- A lean foundation for building higher-level caching, memoization, or coalescing logic with reduced lock contention.

What it isn’t:
- A general-purpose cache replacement. It does not implement a cache eviction policy or persistent storage.
- A complete distributed system. It runs in-process only, coordinating local goroutines.
- A silver bullet for all concurrency problems. It helps with specific hot-path deduplication scenarios.

---

## Key concepts and design decisions

- Shards: The core idea is to split the deduplication surface into multiple groups. Each shard holds its own singleflight.Group instance. This reduces lock contention because only the shard involved in a given key participates in the work.
- Key placement: A user-provided strategy maps a key to a shard. A simple and common method uses a hash function modulo the number of shards. More sophisticated strategies can consider workload patterns, hot keys, or locality.
- Do-like operations: The API mirrors the singleflight pattern—requesters call a Do-like method with a key and a function. The function is executed only once per key per shard, and all concurrent callers share the same result.
- Observability: Instrumentation hooks are included to help you measure contention, shard utilization, and throughput. You can plumb in metrics collectors like Prometheus, OpenTelemetry, or a custom sink.
- Safety and correctness: The implementation preserves the semantics people expect from singleflight – if multiple goroutines request the same key while work is in progress, they all receive the same result and only one call executes the function.

Key benefits:
- Higher aggregate throughput under contention.
- Lower average wait time for hot keys due to distributed load.
- Predictable performance characteristics as you scale shards.

---

## Getting started

This section shows how to bring ShardedFlight into your project, wire up a shard count, and run a simple demo. The end-to-end flow is designed to be straightforward so you can adopt it quickly in real services.

- The repository supports standard Go module workflows.
- It integrates cleanly with existing singleflight-based patterns.

### Installation

To install ShardedFlight, use the standard Go module workflow. Replace v0.x with the latest tag from your releases.

go get github.com/ibrahim-ewd/shardedflight@v0.x.y

Alternatively, you can clone the repository and build from source.

git clone https://github.com/ibrahim-ewd/shardedflight.git
cd shardedflight
go build ./...

Note: The release page contains pre-built artifacts for common platforms. See the Releases page linked at the top of this README for assets you may download and execute in supported environments. The link is also available here: https://github.com/ibrahim-ewd/shardedflight/releases

If you prefer to browse, you can visit the releases page directly: https://github.com/ibrahim-ewd/shardedflight/releases

### Quick start example

The following snippet demonstrates a typical usage pattern. It creates a sharded group with a handful of shards and runs a simple, idempotent operation that should only execute once per key.

Code example:

package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/ibrahim-ewd/shardedflight"
)

func main() {
	// Choose a shard count based on expected load. More shards reduce contention but add memory.
	const shardCount = 8

	sg := shardedflight.New(shardedflight.Options{
		ShardCount: shardCount,
	})

	key := "user:123456"

	// The function should be expensive or I/O-bound; it's executed once per key per shard.
	val, err := sg.Do(context.Background(), key, func() (interface{}, error) {
		// Simulate work
		time.Sleep(100 * time.Millisecond)
		return fmt.Sprintf("result-for-%s", key), nil
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("value:", val)
}

In this example:
- The Do call deduplicates work for the given key across concurrent goroutines.
- The work for a key runs only once per shard that owns that key.
- If multiple goroutines request the same key while work is in flight, they all receive the same result.

Note: The precise API surface may differ slightly depending on the chosen version. The pattern above mirrors the common singleflight flow with an explicit shard parameter.

---

## API overview

ShardedFlight provides an API that mirrors the familiar singleflight pattern but distributes the load across shards. The exact API surface is designed to be ergonomic for Go developers who already use golang.org/x/sync/singleflight.

- New: Create a new ShardedFlight instance with a specified shard count and optional tuning parameters.
- Do/DoChan equivalents: Submit a task for a key and receive the result. If another goroutine asks for the same key while a task is in progress, the caller waits for the existing result.
- In-flight lookup: Inspect in-flight keys per shard for debugging or monitoring.
- Tuning knobs: Adjust shard count, timeouts, and cancellation behavior for complex workloads.
- Observability: Optional hooks for metrics and traces.

Note: The library aims for a simple mental model. You still map keys to shards, and the system guarantees deduplicated execution where multiple callers request the same key concurrently.

---

## Deep dive into architecture

Below is a high-level view of the internal structure. The diagram focuses on the core components and how they interact during a typical call.

<svg width="1000" height="480" viewBox="0 0 1000 480" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="ShardedFlight architecture diagram">
  <defs>
    <linearGradient id="g1" x1="0" x2="1" y1="0" y2="0">
      <stop stop-color="#4F46E5" offset="0"/>
      <stop stop-color="#06B6D4" offset="1"/>
    </linearGradient>
    <style>
      .title { font: 700 16px/1.2 sans-serif; fill: #111; }
      .sub { font: 12px/1.4 sans-serif; fill: #333; }
      .box { fill: #fff; stroke: #888; stroke-width: 2; rx: 6; }
      .header { fill: url(#g1); }
      .bar { fill: #e8f0fe; }
      .arrow { fill: #8b5cf6; }
    </style>
  </defs>

  <!-- Title -->
  <text x="20" y="28" class="title">ShardedFlight Architecture</text>

  <!-- Shards -->
  <rect x="60" y="60" width="170" height="320" class="box"/>
  <text x="75" y="78" font-family="sans-serif" font-size="12" fill="#111">Shard 0</text>

  <rect x="260" y="60" width="170" height="320" class="box"/>
  <text x="275" y="78" font-family="sans-serif" font-size="12" fill="#111">Shard 1</text>

  <rect x="460" y="60" width="170" height="320" class="box"/>
  <text x="475" y="78" font-family="sans-serif" font-size="12" fill="#111">Shard 2</text>

  <rect x="660" y="60" width="170" height="320" class="box"/>
  <text x="675" y="78" font-family="sans-serif" font-size="12" fill="#111">Shard 3</text>

  <!-- In-flight area -->
  <rect x="860" y="60" width="120" height="320" class="box"/>
  <text x="875" y="78" font-family="sans-serif" font-size="12" fill="#111">In-flight</text>

  <!-- Arrows from clients to shards -->
  <polyline class="arrow" points="230,100 260,100" stroke-width="4" fill="none"/>
  <polyline class="arrow" points="430,170 460,170" stroke-width="4" fill="none"/>
  <polyline class="arrow" points="630,240 660,240" stroke-width="4" fill="none"/>

  <!-- Labels -->
  <text x="50" y="420" class="sub">Multiple goroutines issue Do-like calls</text>
  <text x="250" y="440" class="sub">Key mapping to shard</text>
  <text x="450" y="460" class="sub">One deduplicated function per key</text>
  <text x="750" y="420" class="sub">Lock-free coordination inside shard</text>

  <!-- Legend -->
  <rect x="20" y="350" width="170" height="90" fill="#f7f7fb" stroke="#ddd" rx="6"/>
  <text x="28" y="370" font-family="sans-serif" font-size="12" fill="#333">Legend</text>
  <circle cx="40" cy="392" r="6" fill="#10b981"/>
  <text x="50" y="395" font-family="sans-serif" font-size="12" fill="#333">Goroutines</text>
</svg>

This diagram illustrates the core idea: work flows from multiple clients into shards, where each shard maintains a local singleflight-group. Requests for the same key collide in the same shard, ensuring deduplication without global locks.

Key components:
- ShardRouter: Maps a key to a shard using a hash function. It balances keys across shards and minimizes hot spots.
- ShardGroup: An instance of golang.org/x/sync/singleflight-like behavior scoped to a single shard. It holds a group, tracks in-flight requests, and deduplicates concurrent calls for its keys.
- Coordinator: A lightweight orchestrator that orchestrates across shards. It provides the Do-like interface and routes calls to the correct shard.

Design choices:
- Localized contention: By isolating work to a shard, contention remains localized, enabling higher concurrency across the entire system.
- Predictable behavior: Each shard behaves like a standard singleflight group, preserving expected semantics while improving aggregate throughput.
- Modularity: The shard count can be tuned to fit the workload, hardware, and Go scheduler characteristics without rewriting application logic.

---

## Usage patterns and examples

ShardedFlight is designed to slot into code that already uses singleflight. Here are common patterns you can adopt, along with practical notes.

- Simple caching with deduplication: When a request for a particular resource would otherwise trigger a heavy computation or I/O, you can deduplicate by key. The same key invoked concurrently will share the in-flight work.
- Expensive batch operations: If you have a batch process that consolidates multiple inputs into a shared result, you can wrap the batch key in a sharded group to ensure only one batch run is in flight per key.
- Local aggregation: If you need to aggregate results from multiple sources, you can route keys by a domain (user ID, resource ID, region) to a shard that owns the domain, enabling better data locality.

Code snippet: crafting a small in-memory example

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ibrahim-ewd/shardedflight"
)

func main() {
	// Choose an appropriate shard count for the workload.
	const shards = 4

	sf := shardedflight.New(shardedflight.Options{
		ShardCount: shards,
		// You can add more options here, like timeouts or metrics hooks.
	})

	// Demo keys that simulate user IDs or resource IDs.
	keys := []string{"user:1", "user:2", "config:global", "resource:42"}

	// Simulate concurrent callers.
	done := make(chan struct{}, len(keys))
	for _, k := range keys {
		go func(key string) {
			defer func() { done <- struct{}{} }()

			val, err := sf.Do(context.Background(), key, func() (interface{}, error) {
				// Simulate a heavy computation or I/O-bound work.
				time.Sleep(150 * time.Millisecond)
				return fmt.Sprintf("value-for-%s", key), nil
			})
			if err != nil {
				fmt.Printf("error for %s: %v\n", key, err)
				return
			}
			fmt.Printf("result for %s: %v\n", key, val)
		}(k)
	}

	// Wait for all goroutines to finish.
	for i := 0; i < len(keys); i++ {
		<-done
	}
}

In real code, the Do method can return a typed value, and you can adapt the example to your caching layer, memoization, or request coalescing logic.

Usage considerations:
- One function per key: The function you pass should be side-effect free apart from its intended work. Avoid fragile assumptions about the function's execution time; consider cancellation and context propagation.
- Timeouts and cancellation: The Do-like API should propagate cancellation. If you use a context, ensure downstream calls also honor the context.
- Serialization and marshaling: If the work involves I/O or network calls, you can keep it simple by returning plain values or use interface{} with proper type assertions where appropriate.

Advanced usage:
- Per-key config: If certain keys require different timeouts or retry semantics, you can inject a lightweight wrapper that handles per-key configuration before passing the key to ShardedFlight.
- Observability hooks: Integrate with OpenTelemetry or Prometheus by instrumenting shard-level metrics such as in-flight counts, wait times, and success rates.

---

## Tuning and best practices

Tuning ShardedFlight to your workload is a matter of balancing contention, memory, and scheduler behavior. Here are practical guidelines to help you optimize performance.

- Determine the right shard count: Start with a small number of shards and increase gradually while monitoring contention and latency. Too few shards can lead to hot spots; too many shards can add overhead without tangible gains.
- Choose a stable hash strategy: A simple, fast hash function like FNV-1a or xxHash can provide even distribution with low overhead. If your workload has known locality (certain keys cluster), you may tune the mapping to improve cache locality.
- Balance memory usage: Each shard maintains a separate group. Ensure total memory usage stays within your limits. If you observe excessive memory, reduce the shard count or the payload size of the returned results.
- Profile and trace: Use Go’s pprof and tracing tools to identify contention hotspots. Instrument shard-level in-flight counts and latency percentiles to guide tuning decisions.
- Avoid long-running functions in the critical path: If your function blocks for long periods, the benefit of sharding may be limited by downstream bottlenecks. Consider moving heavy work to background tasks or using streaming results where possible.
- Combine with other patterns: ShardedFlight shines when combined with per-key caches, result caching, or rate-limiting schemes. Use it alongside caches to dampen repeated work and improve overall throughput.

---

## Performance and benchmarks

We designed ShardedFlight with throughput in mind. While exact numbers depend on your hardware, Go version, and workload mix, the following general observations guide expectations:

- Under high contention, throughput scales with the number of shards up to a practical limit. Doubling shards does not always double throughput; diminishing returns occur when contention is already low.
- Latency for a hot key tends to drop as shard count and local in-flight handling isolate work from other keys.
- Overhead remains small for typical workloads. The extra coordination is offset by reduced lock contention and faster parallel execution across shards.

Sample benchmark scenario:
- 8 shards
- 100 concurrent goroutines issuing Do for the same set of keys
- Each invocation executes a simulated 100ms I/O-bound operation

Expected outcomes:
- Faster completion overall compared to a single-group setup.
- Consistent latency distribution across keys, with reduced tail latency for hot keys.
- Lower CPU contention in multi-core environments.

These figures are indicative. For real numbers, run your own benchmarks in your target environment, using Go's benchmarking tools and the included instrumentation hooks.

---

## Testing and reliability

Reliability is critical for systems that rely on deduplication. ShardedFlight includes test suites that validate correct deduplication semantics, shard routing, and behavior under concurrent access.

- Unit tests cover:
  - Basic Do correctness and return values
  - Concurrent Do calls for the same key produce a single function invocation
  - Correct shard distribution across a variety of keys
  - Timeouts and cancellation behavior
- Integration tests cover:
  - Typical usage patterns with mock heavy workloads
  - Observability hooks and metric emission
- Property-based tests (where applicable) ensure invariants hold under randomized input patterns
- Stress tests simulate bursty traffic to observe resiliance and stability

When running tests, you can use:
go test ./...

If you need to reproduce production-like load, consider using a load-testing tool that can generate concurrent Do calls across multiple keys and simulate realistic traffic mixes.

---

## Extending and customizing

ShardedFlight is designed to be adaptable. You can extend it in several directions without touching core logic.

- Custom shard routing: Implement your own ShardRouter to influence how keys map to shards. This is helpful when you have domain-specific locality or traffic patterns.
- Alternative backends for results: If your Do function returns complex structures or requires serialization, you can adapt your data model accordingly. Consider JSON or protocol buffers if you persist results.
- Instrumentation adapters: Add hooks to export metrics via Prometheus, OpenTelemetry, or a custom sink. The architecture makes it straightforward to hook into existing monitoring systems.
- Caching layers: Combine ShardedFlight with a per-shard cache to avoid repeated expensive work across restarts or hot keys.

Backward compatibility note:
- The API is designed to be stable across minor versions. If you plan a breaking change, you should communicate it clearly and provide a migration path. The shard routing logic and Do-like semantics should remain intuitive for existing users.

---

## Roadmap and future work

The project roadmap focuses on practical improvements that align with real-world workloads.

- Expand configurability: Expose more knobs for per-shard timers, cancellation strategies, and fan-out controls.
- Enhanced observability: Add richer metrics, traces, and dashboards out of the box.
- Smart key distribution: Develop adaptive shard mapping that rebalances keys in response to traffic shifts.
- Cross-service cooperation: Explore patterns to coordinate deduplication across services while maintaining per-process isolation.
- Advanced caching integration: Ship built-in adapters for common caches to streamline adoption.

If you want to shape the roadmap, contribute issues and pull requests to discuss proposed changes.

---

## Community and contribution

ShardedFlight is an open-source project. Your contributions make the project stronger and more useful to others.

How you can help:
- Report issues with reproducible test cases.
- Propose enhancements via pull requests.
- Write documentation improvements, examples, or benchmarks.
- Help review code and test results.

Contribution guidelines:
- Follow the Go standard formatting and linting conventions.
- Add tests that cover new functionality or edge cases.
- Keep changes focused and well-documented.
- Update the README with usage notes when you add new features.

If you want to learn more about the project’s governance and collaboration approach, check the repository's CONTRIBUTING.md if available.

Releases are published to the official GitHub Releases page, which hosts your binary assets and changelogs. For access to the latest artifacts, visit the Releases section. The link is available here: https://github.com/ibrahim-ewd/shardedflight/releases

---

## FAQ

- What problem does ShardedFlight solve?
  It reduces global lock contention during high-load, concurrent deduplication scenarios. It distributes the work across shards while preserving the deduplication semantics you expect from singleflight.

- How do I choose the number of shards?
  Start with a small number and increase if you observe contention. Monitor in-flight counts, latency, and throughput. The right value depends on your workload and hardware.

- Can I use ShardedFlight with existing singleflight-based code?
  Yes. You can adapt your Do calls to route by a shard key. It helps if your code already uses a Do-like interface.

- Are there any caveats?
  The architecture introduces more moving parts. It’s important to monitor and tune shard counts and ensure your Do function is robust to cancellation and timeouts.

- How do I contribute?
  Open issues to discuss ideas, and submit PRs with tests and documentation updates. See the repository’s CONTRIBUTING guidelines for details.

- Where can I find examples?
  The repository includes examples, and the community can contribute more patterns to demonstrate real-world usage.

---

## License

ShardedFlight is released under a permissive license. See the LICENSE file in the repository for details. If you’re using this in a project with strict licensing requirements, please review the license terms and ensure compliance.

---

## Releases

The project provides release assets that you can download and execute in supported environments. Access the Releases page to grab the latest artifacts, review changelogs, and obtain binaries for your platform. The link is: https://github.com/ibrahim-ewd/shardedflight/releases

For convenience, you can also click the badge above to jump to the release assets. The releases page contains binaries and installation instructions tailored to common environments. If you prefer to browse, visit the releases page directly: https://github.com/ibrahim-ewd/shardedflight/releases

---

## Visual appendix: how to read the architecture diagram

- Shards hold their own singleflight groups. Each shard operates independently.
- Clients submit Do-like requests that are routed to the correct shard based on the key.
- When multiple goroutines request the same key in parallel, only one execution occurs per shard, and the result is shared among all waiting callers.
- The in-flight area represents active Do operations that have not yet completed. Other components may monitor or sample this for metrics.

This mental model helps you reason about latency distribution and throughput. It also clarifies why tuning the shard count matters: it directly influences how many parallel in-flight operations can proceed without stepping on each other’s toes.

---

## The releases page usage note

The project hosts pre-built assets for common platforms on the official Releases page. You can download an appropriate release artifact and execute it in a trusted environment to bootstrap your setup. Access the releases page here: https://github.com/ibrahim-ewd/shardedflight/releases

Additionally, the page provides version history, changelogs, and notes about deprecations or API changes. This information helps you plan migrations and understand the evolution of the project over time.

---

If you want more details or specific examples tailored to your use case, I can tailor the README further.