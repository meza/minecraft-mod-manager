# Rendering performance

Read this guide when investigating or changing rendering time, allocation,
list measurement, or caching. Preserve the ownership boundaries in
[SKILL.md](../SKILL.md). For delivery gates, read [CI and delivery](ci-and-delivery.md).

## Measure a representative interaction

First reproduce the problem with controlled data: a long result list, expanded
markdown, a large highlighted diff, or repeated narrow/wide resize. Record the
data size, viewport, theme, and operation. Compare the same scenario after the
change.

For a project that has a `BenchmarkResultsRender` benchmark in
`internal/ui/search`, run from its module root:

```console
go test ./internal/ui/search -run '^$' -bench '^BenchmarkResultsRender$' -benchmem -count=5
go test ./internal/ui/search -run '^$' -bench '^BenchmarkResultsRender$' -cpuprofile cpu.out -memprofile mem.out
go tool pprof -top cpu.out
go tool pprof -alloc_space -top mem.out
```

Replace the package and benchmark with the real project names. These commands
produce local diagnostic files; keep them outside published source artifacts.
Inspect time and allocations together: fewer allocations do not establish a
faster user interaction. A render benchmark also excludes backend latency and
root routing unless it explicitly exercises them. The Go documentation explains
[benchmark measurement](https://pkg.go.dev/testing#hdr-Benchmarks) and
[profiling](https://go.dev/doc/diagnostics#profiling).

Measure cold rendering separately from repeated rendering, scrolling, and
resize. A warm-cache benchmark alone can conceal a costly invalidation path.
Keep fixture construction outside the timed section unless construction is the
operation being measured.

## Ask only for the layout information you need

An exact total height may require wrapping every item. A boolean overflow check
can stop as soon as the accumulated height exceeds the available rows.

| Caller needs | Suitable computation | Cost implication |
| --- | --- | --- |
| Whether a scrollbar or overflow hint is needed | Stop after proving overflow | Avoids measuring the unseen tail |
| Exact scrollbar thumb size | Exact total height, reused while valid | May require all items |
| The visible page | Render visible items and necessary overscan | Cost follows viewport rather than collection size |
| A changed width | Recalculate width-dependent wrapping | Old height and render caches are invalid |

For an existing list abstraction, reuse its bounded query or viewport support.
Do not add another general list framework. Crush supplies a concrete example:
its `TotalHeight` measures every item, while `Overflows` answers the cheaper
question. It also memoizes theme-dependent syntax styles and lexer selection.
These names belong to Crush, not Bubbles' public list API.
[Crush's render-path guidance](https://github.com/charmbracelet/crush/blob/main/internal/ui/AGENTS.md#common-gotchas)
shows why this distinction matters.

For long lists, prepare syntax-highlighting configuration once for a resolved
theme. Reuse the selected lexer for unchanged language inputs according to the
library's concurrency contract. Do not recreate a Chroma style and repeat lexer
discovery for every visible row on every frame. Content tokenization still needs
to change when the content changes.

## Make cache dependencies visible

Cache only a measured expensive operation. The following local excerpt shows a
single-entry cache around a pure markdown renderer. The application supplies
`renderMarkdown`; its output depends only on the arguments listed here. This is
an example of cache ownership, not an instruction to build a cache library.

```go
type markdownKey struct {
    body          string
    width         int
    themeRevision uint64
    expanded      bool
}

type markdownCache struct {
    valid   bool
    key     markdownKey
    content string
}

func (cache *markdownCache) Render(
    body string,
    width int,
    themeRevision uint64,
    expanded bool,
    renderMarkdown func(string, int, uint64, bool) string,
) string {
    key := markdownKey{
        body: body, width: max(0, width),
        themeRevision: themeRevision, expanded: expanded,
    }
    if !cache.valid || cache.key != key {
        cache.content = renderMarkdown(
            key.body, key.width, key.themeRevision, key.expanded,
        )
        cache.key = key
        cache.valid = true
    }
    return cache.content
}
```

The renderer supplied to this cache must remain the same function for the
cache's lifetime. Its theme revision identifies immutable resolved styles.
If output also depends on selection, search highlighting, color profile,
language, or terminal features, add those dependencies or keep that decoration
outside the cached layer. A revision token works only when every relevant
change advances it. The explicit validity flag prevents a zero-value key from
being mistaken for a populated entry.

This cache belongs to one component and is used on the UI goroutine. It retains
one body and rendered result; an unbounded map would retain every historical
width and content combination. Verify a hit with identical inputs and a miss
for each changing dependency. Also compare cached and uncached output for the
same state.

During rapid resize, width changes invalidate wrapping repeatedly. If profiling
still shows unacceptable work, consider the project's existing strategy for
deferring expensive exact scrollbar calculation until resize settles. Preserve
usable content and correct final geometry. Incremental prewarming and temporary
scrollbar suppression are advanced responses to evidence, not baseline
requirements for every component.
