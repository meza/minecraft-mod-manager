# list

Show which mods you have configured and whether their files are in your mods folder. Use this to confirm what is installed before you update or deploy.

Quick check:

```
mmm list
```

You see a check mark when a mod has a matching lock entry and the file exists; a cross means the lock entry is missing or the file cannot be found. If no mods are configured you will see a short notice instead.

If the lock file is missing, the command treats every mod as not installed and keeps going. Run `mmm install` to populate the lock file and download any missing mods.
