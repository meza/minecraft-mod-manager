# Root routing tests

Read this when designing, building, reviewing, or testing root input routing,
focus, modal precedence, resize, lifecycle, or result ownership.

If you have not selected the applicable test layers for the current task, read
[the testing guide](testing.md) first. Read every applicable guide it selects.

## Test the root's routing decisions

Use distinct initial states and identifiers so accidental broadcasting is
observable. Choose cases relevant to the parent being changed:

| Input or transition | Expected recipient | Check |
| --- | --- | --- |
| Key while modal is open | Modal, except the root's explicit global policy | Editor and underlying list remain unchanged |
| Key with editor focus | Editor | List selection remains unchanged |
| List movement with list focus | Results operation | Editor value remains unchanged |
| Background load result | Matching owner/request | Works even when that owner lacks input focus |
| Obsolete load result | Nobody | Current data, loading, and error remain unchanged |
| Widget tick | Owning widget lifecycle | Returned next command survives without broadcasting input |
| Resize | Root allocation, then each affected child once | Correct child rectangle and preserved command |
| Mouse at a component edge | Region containing the cell | Local coordinates use actual layout origin |
| Owner replacement | New instance, old supported work cancelled | Old results and recurring work do not revive old state |
| Focus transfer | Previous blur and next focus | One focused visible cursor; focus command survives |

Use a sentinel message from a child command to detect command loss. Do not assert
that a command is merely non-nil when delivering its result is what matters.

