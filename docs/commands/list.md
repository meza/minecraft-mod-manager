# list

Show which mods you have configured and whether their files are in your mods folder. Use this to confirm what is installed before you update or deploy.

Quick check:

```
mmm list
```

You see a check mark when a mod has a matching lock entry and the file exists with the expected hash; a cross means the lock entry is missing, the file cannot be found, or the hash does not match. When output is not colorized, the command uses V for installed mods and X for missing mods. If no mods are configured you will see a short notice instead.

Each entry includes the platform in square brackets. When the hash does not match, the output calls that out and suggests running `mmm install` to fix it. If unmanaged jar files are detected, the command prints a notice after the list and recommends running `mmm scan`.
