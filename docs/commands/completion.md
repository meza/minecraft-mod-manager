# Shell completion

This guide describes the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it.

Use `completion` to generate an integration script for your shell. Completion helps discover commands, flags and supported value suggestions without initializing an MMM installation.

## Usage

```shell
mmm completion bash
mmm completion zsh
mmm completion fish
mmm completion powershell
```

Each command emits the script for the named shell. To save it, redirect the output to a file. For example:

```shell
mmm completion bash > mmm-completion.bash
```

Use the generated script with your shell's completion setup. Generating a script does not install it into your shell configuration.

## Options

| Option | Default | Effect |
| --- | --- | --- |
| `--no-descriptions` | `false` | Omit descriptions from completion suggestions. |

For example:

```shell
mmm completion fish --no-descriptions
```

## Suggestions and setup

Init supplies static value suggestions for `--loader` and `--release-types`. Those suggestions require neither network nor configuration access. They are separate from Minecraft version suggestions inside interactive init, which can use Minecraft metadata.

Script generation and completion requests do not prompt for setup or modify installation metadata. Failure to register optional completion support must not prevent ordinary init use.

See [init](init.md), [help](help.md) and [shared command behavior](README.md).
