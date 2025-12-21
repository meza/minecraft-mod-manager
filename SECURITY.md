# Security Policy

The code is continuously scanned for known vulnerabilities and the dependencies are automatically kept up-to-date.
To make sure that you have the latest security updates, please make sure to always use the latest version of the Minecraft Mod Manager.

There will be limited support for major versions going forward but please upgrade as soon as you can.

## Reporting a Vulnerability

To report a vulnerability, please open an issue in the [issue tracker](https://github.com/meza/minecraft-mod-manager/issues).

## Configuration Trust Boundary

MMM treats `modlist.json` as trusted input and uses it to decide where it reads, writes, and deletes mod files.

In particular, `modsFolder` can be an absolute path or can point outside the folder that contains `modlist.json`. This is
useful for server administrators and custom setups, but it also means you should only run MMM against `modlist.json` files
you trust.
