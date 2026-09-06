# Rendering tests

Read this when designing, building, reviewing, or testing rendering, terminal
geometry, cursor output, deterministic fixtures, or goldens. The Results
example uses the [worked component](results-example.md).

If you have not selected the applicable test layers for the current task, read
[the testing guide](testing.md) first. Read every applicable guide it selects.

Before using or reviewing the Results render test example, read the
[worked component](results-example.md), including its input, style, and ownership
assumptions, if you have not already read it.

## Make rendering reproducible

Pin all inputs that can affect output:

| Input | Set it at |
| --- | --- |
| Width and height | Direct `Render` arguments or runtime terminal options |
| Semantic styles and dark/light choice | Fresh fixture constructor |
| Color profile | Runtime program option; explicit style policy for direct rendering |
| Time and relative date labels | Injected clock or fixture timestamp |
| Ordering | Fixed fixture order or production deterministic sort |
| Random identifiers | Injected generator or stable fixture values |
| Locale and terminal features | Boundary configuration, not hidden render-time detection |

For the `Results` component, add a `strings` and `github.com/charmbracelet/x/ansi`
import and check cell bounds directly. This test excerpt assumes `component` is
a fresh results instance containing long Unicode labels:

```go
for _, dimensions := range [][2]int{{0, 0}, {1, 1}, {8, 2}, {40, 6}} {
    width, height := dimensions[0], dimensions[1]
    rendered := component.Render(width, height)
    if rendered != component.Render(width, height) {
        t.Fatal("render changed without an input change")
    }
    if width == 0 || height == 0 {
        if rendered != "" {
            t.Fatalf("zero allocation rendered %q", rendered)
        }
        continue
    }
    lines := strings.Split(rendered, "\n")
    if len(lines) > height {
        t.Fatalf("render has %d lines, allocation is %d", len(lines), height)
    }
    for _, line := range lines {
        if ansi.StringWidth(line) > width {
            t.Fatalf("line exceeds %d cells: %q", width, line)
        }
    }
}
```

Use actual semantic styles in additional cases so ANSI sequences are present.
Include wide characters, combining marks, long words, and frame variants in the
components that own frames. When testing real cursor metadata, separately
assert its local and composed coordinates, visibility, and clipping.

For goldens, compare direct component output when possible. This avoids making
terminal repaint control sequences part of every visual expectation. Pair a
golden with a state assertion: identical pixels cannot prove the intended result
was selected or that a save succeeded.

Choose line-ending policy according to what is stored. For ordinary LF text
fixtures, use `*.golden text eol=lf` in `.gitattributes`. For fixtures preserving
exact terminal bytes, use a more specific path with `-text` to disable conversion.
Do not blindly convert byte fixtures or strip ANSI if color is the contract under
test. Update only the affected expectation through the project's golden tool,
inspect the diff, and rerun comparison without update mode. Teatest's
`RequireEqualOutput` supports `-update` and currently uses the system `diff`
utility; account for that dependency in the project's test setup.
