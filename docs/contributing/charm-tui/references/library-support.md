# Published-library support

Read this guide when working in a published component library or preparing one
for publication. This includes its supported toolchain, dependency contract,
and public support documentation.

For a published component library, maintain a versioned support matrix in its
public documentation instead of saying only "supports Charm". Record the library
release, minimum supported Go version, Bubble Tea/Bubbles/Lip Gloss major versions,
Bubblebook v2, and the pinned experimental teatest dependency behind the local
test setup. This is a durable consumer contract, not merely a handoff note.

The research's illustrative matrix has this shape; its library and Go versions
are example values, not dependency selections for a new project:

```text
Component library v0.8.x
Go:          1.24+
Bubble Tea:  v2
Bubbles:     v2
Lip Gloss:   v2
Bubblebook:  v2
teatest:     x/exp; wrapped by internal test harness
```

Populate the maintained record from the actual supported module graph and
toolchain, and keep it accurate as library releases change those requirements.
