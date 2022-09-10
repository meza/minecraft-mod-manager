# 2. Console Log/Error only in Actions

Date: 2022-09-10

## Status

Accepted

## Context

Javascript allows us to do a lot of things. One of the things we can do is to
write to the console from any place within the codebase. This is very useful
for debugging and development. However, we should not do this in production.

With a CLI application we need to be very clear about what we are communicating
with the user and how we're allowing them to interact with the application.

## Decision

To make sure that we communicate with the user on the right level,
all invocations to the `console.log` and the `console.error` functions should
be done in the `actions` folder. This means that the `console.log` and the
`console.error` functions should not be used in the `lib` folder.

To signal errors to the code, we should always throw exceptions and handle them
appropriately in the actions. There we can decide how to communicate the error
to the user and how to allow them to handle it.

### debugging

`console.debug` is allowed at any point in the code as long as it's protected by the
global debug option coming from the action.

## Consequences

We end up with a clear separation between where we communicate with the user and where
we communicate with the developer. This allows us to deal with the user IO in area.
