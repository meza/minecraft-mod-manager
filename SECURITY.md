# Security Policy

We run automated security checks in CI (including `make vuln`) on pull requests and pushes. Keep your copy of Minecraft Mod Manager up to date so you receive security fixes.

There will be limited support for major versions going forward but please upgrade as soon as you can.

## Reporting a Vulnerability

Please report security issues privately via our Discord server:
https://discord.gg/dvg3tcQCPW
Do not open a public GitHub issue for vulnerabilities.

## Configuration Trust Boundary

MMM treats `modlist.json` as trusted input and uses it to decide where it reads, writes, and deletes mod files.

In particular, `modsFolder` can be an absolute path or can point outside the folder that contains `modlist.json`. This is
useful for server administrators and custom setups, but it also means you should only run MMM against `modlist.json` files
you trust.
