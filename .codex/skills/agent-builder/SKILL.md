---
name: Agent Builder
description: How to create reasoning agents.
---

# Agent Builder Instructions


## Process

- Use the instruction-engineer agent to craft the instruction sets for new reasoning agents.
- When ready, place the instruction set files in the `./agents/[agent-name].md` file.

## Anatomy of an Agent configuration file [agents/{agent-name}.md]

Every agent consists of:

- **Frontmatter** (YAML): Contains `name` and `description` fields. These are the only fields that Claude reads to determine when the skill gets used, thus it is very important to be clear and comprehensive in describing what the skill is, and when it should be used.
- **Body** (Markdown): Instruction set created in step 1.
