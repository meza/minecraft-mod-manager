# Terminal recordings

Read this guide when creating or reviewing a terminal demonstration for
requested documentation or visual review. The shared scope and verification
rules in [SKILL.md](../SKILL.md) apply.

## Record a focused demonstration

Use a recording to explain a specific interaction already covered by state and
render assertions. The example tape below assumes the project's
`go run ./cmd/bubblebook` starts its catalogue and displays the word `Search`.
Replace that command and readiness text with the project's documented entry.

```text
Require go
Output search-catalogue.gif
Set Width 1000
Set Height 600
Set FontSize 20
Hide
Type "go run ./cmd/bubblebook"
Enter
Wait /Search/
Show
Sleep 2s
```

Save the tape beside the project's demonstration material and run
`vhs search-catalogue.tape`. `Wait` observes readiness; `Sleep` here sets the
viewing pace after readiness, not an application test assertion. Add keys only
for the catalogue's actual navigation bindings. VHS dimensions are recording
pixels, not the component's width and height in terminal cells. Pin the font,
theme, fixture, and tool setup when visual comparability matters. See the
[VHS command reference](https://github.com/charmbracelet/vhs#vhs-command-reference).

A successful tape records what appeared. It does not establish correct stale
result rejection, command propagation, or focus ownership. Keep those checks in
the [testing layers](testing.md), and inspect the named component states using
the [Bubblebook workflow](bubblebook.md).
