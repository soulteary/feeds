# Performance tests and XML extension optimization

## Scope and fixtures

`performance_benchmark_test.go` contains repeatable benchmarks for 10, 100 and
1,000 shared Feed items: Atom/RSS/JSON output, Atom writer output to `io.Discard`,
and decoded-model conversion back to Feed. Each item has a stable explicit ID,
a link, a fixed timestamp and short HTML content. Fixtures are constructed before
timing; conversion, serialization and returned-result allocations are timed.
These are synthetic workloads rather than production traffic or a concurrency test.

`BenchmarkXMLExtensions` serializes 100 Atom entries, each with a foreign metadata
element, an attribute and a nested text element. Fixture construction is excluded
with ResetTimer. The benchmark keeps the returned string alive. It is specifically
a test of XML extensions; it does not claim the same gains for feeds without them.

## Reproduction

```sh
go test -run '^$' -bench 'BenchmarkFeedFormats|BenchmarkXMLExtensions' -benchmem -count=5 -cpu=1
```

For the focused samples checked into this directory:

```sh
GOMAXPROCS=1 go test -run '^$' \
  -bench 'BenchmarkXMLExtensions|BenchmarkFeedFormats/(ToAtom|ToRSS|ToJSON)/1000$' \
  -benchmem -benchtime=300ms -count=5 -cpu=1
```

To compare revisions, copy the same benchmark file to separate clean checkouts
of the baseline and optimized source; run them sequentially with the same Go,
CPU settings and power state. Do not run tests, profiles or other benchmarks
concurrently with timed measurements. Check bytes and allocation counts as well
as latency, and use more samples/benchstat for decisions based on small differences.

The final baseline is `643e53b` (the main merge of RSS/Atom conversion PR #14).
The baseline and final optimization samples use Go 1.27.3, darwin/arm64, Apple M5,
`GOMAXPROCS=1`, five samples and a 300ms minimum per sample. The optimized revision
is the commit adding this document; only the XML extension implementation is
changed in production code. Timing includes GC effects; B/op is allocated bytes
per operation, not live heap or peak RSS. Allocations/op is an allocation count.

Raw samples: [baseline](performance/baseline-go1.27.3.txt),
[optimized](performance/optimized-go1.27.3.txt).

## Results

For the 100-entry extension fixture, allocated bytes fell from approximately
1,503,615 B/op to 881,208 B/op (41.4% less), and allocation count fell from
6,340 to 3,440 per operation (45.7% fewer). Ordinary 1,000-item Atom/RSS/JSON
controls retained their original allocation counts and essentially identical
allocated bytes, as expected when those fixtures contain no custom extensions.

Latency was noisy even for unchanged control paths. The sequential five-sample
extension medians were 0.933 ms and 0.488 ms, but the unchanged JSON control also
appeared 43.5% faster, so that timing change cannot be attributed entirely to the
patch. An additional [interleaved run](performance/interleaved-go1.27.3.txt)
using precompiled binaries in baseline/optimized/optimized/baseline/baseline/optimized
order had medians of 0.731 ms and 0.530 ms, with individual outliers in both groups.
The allocation reduction is repeatable in these samples; no fixed throughput
improvement is claimed. Workloads with Unicode names, newline-bearing leaves or
mostly mixed XML content use the fallback more often and may gain less.

## Implementation and compatibility

- Accept known-valid ASCII XML names without allocating a decoder. All other
  names use the original decoder path, preserving Unicode and historical edge
  behavior. This is a fast path, not stricter XML name validation or a global cache.
- Avoid constructing an encoder for empty content or one plain-text content part.
  Use the standard library's escaping rules. Newline-containing text stays on the
  original encoder path because EscapeText and EncodeToken spell LF differently.
- Retain the existing encoder for elements, comments and mixed content. Do not
  alter indentation, namespace validation, ordering, source ownership or public API.
- Preserve empty writes and short-write errors using a small writer adapter.
  No buffer pools, unsafe conversions, shared mutable cache or dependency is added.

Differential fuzz tests compare names and escaped content against the previous
implementation. They cover Unicode, invalid UTF-8, XML metacharacters, entity-like
strings and CR/LF/tab text. Existing tests cover namespace conflicts, mixed content,
invalid structures and failed writers. The saved corpus includes an unusual name
found by fuzzing; keeping the slow fallback avoids changing its historical result.
Short local fuzz runs passed; this is finite evidence rather than proof over all
inputs. CI also runs both differential fuzzers briefly on every quality check.

## Initial investigation

The earlier Go 1.27.1 investigation and raw CPU/allocation profiles are preserved
under [performance/exploratory](performance/exploratory/). That investigation used
commit `5cd1f50`, before its main merge, and a broader scratch prototype including
slice preallocation. Its short-leaf implementation was subsequently refined to
preserve writer semantics. Those historical samples are not the final optimization
comparison and should not be mixed with the Go 1.27.3 samples.

The allocation profile showed repeated encoder buffers and name-parser allocations
in extension serialization. CPU samples included substantial runtime/host noise,
so no universal throughput percentage is promised. Generic slice preallocation
saved less than 1% allocated bytes in the tested ordinary feeds and is not adopted.
Replacing ToXML with a strings.Builder also had inconsistent memory/timing results
across formats and is not adopted. Existing Writer APIs can avoid retaining a
complete output string when the consumer writes to a network/file sink; writing
into another growing memory buffer may not provide that benefit. WriteJSON still
uses encoding/json's encoder and is not claimed to stream individual items.
