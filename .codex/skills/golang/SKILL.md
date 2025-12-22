---
name: golang
description: >
  MUST USE when: (1) writing Go/Golang code (.go files), (2) reviewing Go code,
  (3) user asks about Go/Golang coding standards. OVERRIDES default Go/Golang
  language conventions. These project-specific guidelines take precedence over
  common Go idioms and community conventions.
---

# Go Programming Guidelines

## Primary Directive

Before writing or reviewing any Go code, ask:

**"Would this name/pattern be considered good practice in any other programming language?"**

If the answer is yes, use it. This question overrides all Go idioms, community conventions, and "idiomatic Go" guidance. Apply this test to every naming decision, every pattern choice, every code structure.

Traditional software craftsman practices take absolute precedence over Go conventions.

## Naming Conventions

**Clarity over brevity.**

Eliminate single-letter variable names except widely-accepted Go conventions:

| Allowed | Context |
|---------|---------|
| `t` | Testing |
| `err` | Errors |
| `cfg` | Configuration |
| `cmd` | Commands |
| `ctx` | Context |

All other names must be descriptive and meaningful.

**Examples:**

```go
// BAD
func (s *Server) h(w http.ResponseWriter, r *http.Request)

// GOOD
func (server *Server) handleRequest(writer http.ResponseWriter, request *http.Request)
```

```go
// BAD
for i, v := range items {
    process(v)
}

// GOOD
for index, item := range items {
    process(item)
}
```

## Library Selection

Before implementing from scratch:

1. Search https://pkg.go.dev/ for established solutions
2. Evaluate stability and community trust
3. Prefer mature libraries over experimental or newly-released packages

## Implementation Principles

Always return to the primary directive: **Would this be good practice in any other language?**

| Priority | Over |
|----------|------|
| Clarity | Brevity |
| Maintainability | Cleverness |
| Descriptiveness | Convention |
| Readability | Go idioms |

Reject any Go convention that fails the cross-language test. Treat Go as you would any other language.
