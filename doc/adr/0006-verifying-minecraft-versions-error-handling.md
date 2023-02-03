# 6. Verifying Minecraft Versions Error Handling

Date: 2023-02-02

## Status

Accepted

## Context

There is a non-zero chance of the minecraft metadata api not responding when querying the known minecraft versions.

The proposed solutions to mitigate this especially around the verification of user input were the following:

- store the previous successful call's response in a file somewhere
- apply a retry logic
- be lenient and assume it's correct

## Decision

The decision is that the call will go through the rate-limiter facility which does retry a few times but ultimately
if the request fails, we will assume that the entered string is valid.

We're giving the benefit of the doubt to the users that they know which version they're looking for.

Later down the line, we'll implement a string pattern matching algorithm that will attempt to assess if the version string
supplied is a possible version at all. Issue [#241](https://github.com/meza/minecraft-mod-manager/issues/241)

## Consequences

With this change, the validity of the version string won't be as reliably perfect for a cli-only user.
This doesn't change much in the reliability of the application because the config file has always provided a way for people
to overwrite a previously validated game version.

