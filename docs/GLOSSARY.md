# Minecraft Mod Manager glossary

## Authority and use

This is the canonical vocabulary for Minecraft [Mod](#mod) Manager (MMM). It serves operators, contributors and maintainers reading or writing product requirements, command guides, acceptance scenarios and implementation code.

This glossary defines the project's terms independently of any particular requirements document. Definitions describe the Go-port baseline unless explicitly labelled as future concepts; they do not certify that a released build implements the described behaviour. Changing vocabulary must not silently change a product requirement.

Use these terms consistently in new or revised documentation, user-facing language and internal code names. Use the specific entity when a distinction affects behaviour: a [project](#project) is not a [mod config](#mod-config), an [artifact](#artifact) is not a [local file](#file), and a [locked artifact](#locked-artifact) is not proof of installation. “[Mod](#mod)” remains the general domain term defined below; it must not stand in for a more specific entity in a precise contract. Translations must preserve these distinctions.

Keep established CLI flags, configuration keys and external identifiers exactly as specified by their interfaces. Their spelling does not create a competing definition. Existing material and internal names converge as they are revised; this glossary does not itself rename interfaces or certify existing terminology.

Entries are grouped by concept. The alphabetical index includes every defined term. Command procedures and numeric exit codes are documented separately in the [command guides](commands/README.md).

## Alphabetical index

- [Active display](#active-display)
- [Active end](#active-end)
- [Addition](#addition)
- [Adoption](#adoption)
- [Artifact](#artifact)
- [Authorization](#authorization)
- [Business operation](#business-operation)
- [Capability component](#capability-component)
- [Collision](#collision)
- [Command](#command)
- [Command flow](#command-flow)
- [Compatibility check](#compatibility-check)
- [Compatible](#compatible)
- [Configuration](#configuration)
- [Configuration directory](#configuration-directory)
- [Confirmation](#confirmation)
- [Constraint](#constraint)
- [Declared state](#declared-state)
- [Dependency management](#dependency-management)
- [Diagnostics](#diagnostics)
- [Disabled file](#disabled-file)
- [Discovery](#discovery)
- [Display name](#display-name)
- [Drift](#drift)
- [Eligibility](#eligibility)
- [Excluded file](#excluded-file)
- [Failure](#failure)
- [File](#file)
- [Force](#force)
- [Forced termination](#forced-termination)
- [Guided constraint resolution](#guided-constraint-resolution)
- [Ignored file](#ignored-file)
- [Incompatible](#incompatible)
- [Incomplete execution](#incomplete-execution)
- [Inconclusive result](#inconclusive-result)
- [Initialization](#initialization)
- [Inspection](#inspection)
- [Install operation](#install-operation)
- [Installation](#installation)
- [Installation boundary](#installation-boundary)
- [Interactive execution](#interactive-execution)
- [Invalid invocation](#invalid-invocation)
- [Invocation](#invocation)
- [Latest](#latest)
- [Loader](#loader)
- [Lock entry](#lock-entry)
- [Locked artifact](#locked-artifact)
- [Lockfile](#lockfile)
- [Lookup](#lookup)
- [Managed enabled and disabled state](#managed-enabled-and-disabled-state)
- [Managed file](#managed-file)
- [Manifest cache](#manifest-cache)
- [Minecraft target](#minecraft-target)
- [Minecraft version manifest](#minecraft-version-manifest)
- [Mod](#mod)
- [Mod config](#mod-config)
- [Mod identity](#mod-identity)
- [Modlist](#modlist)
- [Mods directory](#mods-directory)
- [No-op](#no-op)
- [Observed installation state](#observed-installation-state)
- [Operation](#operation)
- [Operator](#operator)
- [Partial success](#partial-success)
- [Performance recording](#performance-recording)
- [Permanent transcript](#permanent-transcript)
- [Pin](#pin)
- [Platform](#platform)
- [Project](#project)
- [Project ID](#project-id)
- [Pruning](#pruning)
- [Recognition](#recognition)
- [Reconciliation](#reconciliation)
- [Recovery](#recovery)
- [Redirected execution](#redirected-execution)
- [Release policy](#release-policy)
- [Release type](#release-type)
- [Removal](#removal)
- [Repair](#repair)
- [Resolution](#resolution)
- [Resolution evidence](#resolution-evidence)
- [Resolved desired state](#resolved-desired-state)
- [Retained artifact](#retained-artifact)
- [Retry](#retry)
- [Safe cancellation](#safe-cancellation)
- [Satisfied installation](#satisfied-installation)
- [Settled result](#settled-result)
- [Success](#success)
- [Telemetry](#telemetry)
- [Terminal session](#terminal-session)
- [Transcript commit](#transcript-commit)
- [Unattended execution](#unattended-execution)
- [Unmanaged file](#unmanaged-file)
- [Update](#update)
- [Valid locked artifact](#valid-locked-artifact)
- [Version change](#version-change)
- [Version fallback](#version-fallback)
- [Version string](#version-string)
- [Visible file](#visible-file)
- [Visual primitive](#visual-primitive)

## Identity and installation

### Operator

The person directing MMM's work on an [installation](#installation), directly or through automation. Server operators are the primary audience; players and modpack maintainers use the same capabilities.

### Mod

A Minecraft modification, discussed generally without specifying a [mod config](#mod-config), [platform project](#project), [artifact](#artifact) or [local file](#file). Qualify the entity whenever identity, [lookup](#lookup), ownership or state matters.

### Platform

A service through which MMM identifies [projects](#project) and obtains [artifact](#artifact) metadata and downloads. The baseline supports CurseForge and Modrinth. Use “platform” for this domain role; “source” is broader provenance language, not another [mod identity](#mod-identity).

### Project

A [platform](#platform)-hosted [mod](#mod) listing identified by a [project ID](#project-id) on that [platform](#platform). A project can supply multiple [artifacts](#artifact). It is not the MMM software repository or a local [installation](#installation).

### Project ID

The identifier of a [project](#project) within a [platform](#platform). An ID alone does not establish MMM [mod identity](#mod-identity) without its [platform](#platform). Keep [platform](#platform)-specific identifier syntax intact.

### Mod identity

The pair of [platform](#platform) and [project ID](#project-id). A [display name](#display-name), filename or [version string](#version-string) does not replace that identity. The same [mod](#mod) hosted on two [platforms](#platform) has distinct platform identities; MMM does not silently switch between them.

### Display name

[Platform](#platform)-provided descriptive metadata for a [mod](#mod). A stored name may refresh during [installation](#install-operation) or [update](#update); a manual edit does not make it a protected [operator](#operator) label.

### Artifact

A concrete distributable [mod](#mod) [file](#file) whose identity can be recorded with filename, hash and source information. It can be selected without being downloaded or present locally. An artifact's [display version](#version-string) alone does not establish exact [file](#file) content.

### File

A filesystem entry at a particular path. In [mod](#mod) [operations](#operation), distinguish the local file from the [artifact](#artifact) it is expected to contain: a matching filename does not prove matching content.

### Installation

The [modlist](#modlist)-selected context MMM operates on: the [modlist](#modlist), associated [resolution](#resolution) metadata and the configured [mods directory](#mods-directory) with its actual [files](#file). “Install MMM” means obtaining the application; use “[mod](#mod) installation” when that distinction is needed.

### Installation boundary

The filesystem targets selected through trusted [modlist](#modlist) within which MMM performs its intended [file operations](#operation). This is not necessarily the current working directory: absolute [mods paths](#mods-directory) are supported. It does not authorize following unsafe paths or overwriting protected [collisions](#collision).

### Configuration

The general term for settings. For MMM's [installation](#installation) configuration, use [modlist](#modlist) for the complete document and [mod config](#mod-config) for an individual [mod](#mod) entry. The `--config` option selects the [modlist file](#modlist).

### Configuration directory

The directory containing the selected [modlist file](#modlist). Relative [mods-directory](#mods-directory) paths resolve against it, and `.mmmignore` resides beside the [modlist file](#modlist). It need not be the shell's current directory.

### Modlist

The complete document describing the [operator](#operator)'s desired [installation](#installation), stored by default in `modlist.json`. It contains installation-wide settings—[Minecraft target](#minecraft-target), [loader](#loader), [mods directory](#mods-directory) and default [release policy](#release-policy)—plus the collection of [mod configs](#mod-config). `--config` selects an alternative modlist file. Inclusion in the modlist does not imply that an [artifact](#artifact) has been [resolved](#resolution) or [installed](#install-operation). Use “modlist” for the whole document and “[mod config](#mod-config)” for one [mod](#mod)'s entry.

### Mod config

One entry in the [modlist](#modlist)'s [mod](#mod) collection, identifying a desired [mod](#mod) by its [platform](#platform) and [project ID](#project-id) and recording its per-mod [lookup](#lookup) settings. These can include a [pin](#pin), [release-type override](#release-policy) and [version fallback](#version-fallback) permission. These settings combine with the [modlist](#modlist)'s installation-wide settings to determine the [mod](#mod)'s effective [constraints](#constraint). A mod config is neither a [lock entry](#lock-entry) nor an installed [file](#file).

### Lockfile

The metadata [file](#file) recording exact [resolved](#resolution) [artifacts](#artifact), by default `modlist-lock.json`. It follows the [modlist](#modlist) file's basename: `server.json` uses `server-lock.json`. It records [resolved desired state](#resolved-desired-state), not an automatically [adopted](#adoption) inventory of [local files](#file).

### Lock entry

An [artifact](#artifact)-[resolution](#resolution) record in the [lockfile](#lockfile), including the required identity and [integrity evidence](#resolution-evidence). Its presence alone proves neither that a [file](#file) is installed nor that the entry satisfies the current [modlist](#modlist). Missing entries and corrupt [existing evidence](#resolution-evidence) require different handling.

### Mods directory

The directory containing the [installation](#installation)'s actual [mod](#mod) [files](#file), selected by `modsFolder`. It may contain [managed](#managed-file), [unmanaged](#unmanaged-file) and [excluded files](#excluded-file). [Discovery](#discovery) and [pruning](#pruning) examine immediate [files](#file), not nested directories. “Mods folder” is the interface wording for this same directory.

## Lookup and state

### Lookup

Finding [project](#project) or [artifact](#artifact) information using [platform](#platform) metadata. When looking for an [eligible](#eligibility) [artifact](#artifact), MMM applies the [mod](#mod)'s effective [constraints](#constraint) and the [command](#command)'s policy. For example, [update](#update) looks for a newer [eligible](#eligibility) [artifact](#artifact) for an [unpinned](#pin) [mod](#mod). An identification lookup supports [recognition](#recognition) of an existing [artifact](#artifact) without requiring it to meet the [installation](#installation)'s [constraints](#constraint).

Lookup can establish that a [project](#project) exists without finding an [eligible](#eligibility) [artifact](#artifact). A [failed request](#failure) does not establish absence or [incompatibility](#incompatible). [Discovery](#discovery) finds [local files](#file); [resolution](#resolution) establishes an exact [artifact](#artifact) with the [evidence](#resolution-evidence) needed to record and reproduce it. Lookup alone does not imply [resolution](#resolution), downloading or [installation](#install-operation).

### Constraint

A rule that limits which [artifacts](#artifact) are [eligible](#eligibility) during [lookup](#lookup) for a [mod](#mod). Constraints come from both the [modlist](#modlist)'s installation-wide settings and the individual [mod config](#mod-config).

A [mod](#mod)'s effective constraints combine the installation-wide [Minecraft target](#minecraft-target) and [loader](#loader), the default allowed [release types](#release-type) unless the [mod config](#mod-config) overrides them, and any per-mod [pin](#pin). Per-mod [version fallback](#version-fallback) permission changes the Minecraft-version matching rule; it does not waive unrelated constraints. The [mods directory](#mods-directory) is a location setting, not an artifact-lookup constraint.

### Minecraft target

The Minecraft version requested for [lookup](#lookup) or a [compatibility check](#compatibility-check). The [modlist](#modlist) stores it as `gameVersion`. Explicit targets must be [manifest-listed](#minecraft-version-manifest); accepting a target does not establish [artifact](#artifact) availability.

### Loader

Minecraft mod-loading software used by the [operator](#operator)'s game or server. The [modlist](#modlist)'s `loader` setting identifies the mod-loader project used by that [installation](#installation).

Within MMM, this identity serves as a [lookup](#lookup) [constraint](#constraint): MMM matches it against [platform](#platform) metadata to find suitable [artifacts](#artifact). It does not define a separate MMM product mode or imply a server-only filter.

### Release type

An [artifact](#artifact)'s [platform](#platform) release category: `alpha`, `beta` or `release`. The value `release` here is not a Minecraft version or an MMM application release.

### Release policy

The allowed [artifact](#artifact) [release types](#release-type) used during [lookup](#lookup). `defaultAllowedReleaseTypes` supplies the [installation](#installation) default, initially release-only; a [mod](#mod)'s `allowedReleaseTypes` overrides that default for that [mod](#mod). An override does not modify the [installation](#installation) default.

### Pin

An explicit [platform](#platform)-specific version [constraint](#constraint) that ordinary [updates](#update) preserve. `--version` sets it and `--unpin` clears it. An unpinned [mod](#mod) has no explicit version pin. Modrinth uses its version number; CurseForge uses the [artifact](#artifact) filename. A pin is not a universal [artifact](#artifact) ID, integrity hash or semantic-version range.

### Version string

A [platform](#platform)'s version label for a [mod](#mod) [artifact](#artifact). Do not assume it follows semantic versioning or that a changed label proves newer content. Qualify “version” as Minecraft version, [mod](#mod) version or MMM version when the referent is otherwise unclear.

### Latest

For a default [Minecraft target](#minecraft-target), the latest stable Minecraft release identified by the available validated [manifest](#minecraft-version-manifest). [Cached metadata](#manifest-cache) can make that information stale. For [mod](#mod) [artifacts](#artifact), use “newest [eligible](#eligibility) [artifact](#artifact)” and the applicable [lookup](#lookup) rules rather than implying the Minecraft meaning.

### Version fallback

The per-mod `allowVersionFallback` permission to consider an older Minecraft release's [artifact](#artifact) when a suitable [target](#minecraft-target) [artifact](#artifact) is unavailable. It defaults to disabled and searches nearest earlier releases within the same release series, without crossing series or applying snapshot fallback. This differs from scan's alternate-platform [lookup](#lookup) and [cached-manifest fallback](#manifest-cache).

### Eligibility

Whether an [artifact](#artifact) meets the applicable [platform](#platform)-based [lookup](#lookup) [constraints](#constraint), including permitted [version fallback](#version-fallback). Eligibility does not prove runtime success or the availability of sufficient download and integrity information.

### Resolution

Establishing the exact [artifact](#artifact) to use under the applicable rules, with the [evidence](#resolution-evidence) needed to record and reproduce it. Resolution precedes downloading: a resolved [artifact](#artifact) can still fail to download. [Failure](#failure) to resolve must not be represented by an invented [artifact](#artifact) or incomplete [lock entry](#lock-entry).

### Resolution evidence

The recorded information supporting an exact [resolution](#resolution), including [artifact](#artifact) identity, filename, hash and source information. Missing evidence is not equivalent to corrupt or contradictory evidence. Existing untrustworthy evidence is preserved and reported, not silently reconstructed.

### Locked artifact

An exact [artifact](#artifact) recorded in the [lockfile](#lockfile). “Locked” does not mean [pinned](#pin), downloaded, currently installed or necessarily valid under a subsequently changed [modlist](#modlist).

### Valid locked artifact

A [locked artifact](#locked-artifact) that remains acceptable under the current [modlist](#modlist) and applicable [authorization](#authorization). This includes an explicitly [adopted](#adoption) or [force-retained artifact](#retained-artifact) authorized for a particular target despite platform-reported [incompatibility](#incompatible). That exception does not authorize arbitrary future [artifacts](#artifact) or waive integrity requirements.

### Declared state

The [mods](#mod) and [constraints](#constraint) currently requested in the [modlist](#modlist). It can intentionally remain ahead of [installed files](#file) after a failed download.

### Resolved desired state

The exact [artifacts](#artifact) recorded to reproduce the desired managed [installation](#installation). The [lockfile](#lockfile) owns this information; it can be incomplete or retain [recovery](#recovery) evidence while [reconciliation](#reconciliation) remains unfinished. It is not a claim about current disk contents.

### Observed installation state

What [inspection](#inspection) can establish about actual [files](#file) relative to [mod configs](#mod-config) and [resolution evidence](#resolution-evidence), including missing [files](#file), [missing evidence](#resolution-evidence) and content mismatches. An unknown fact remains unknown; filename matching alone is insufficient.

### Satisfied installation

A managed [installation](#installation) matching the [modlist](#modlist) and its [valid locked artifacts](#valid-locked-artifact). It can coexist with [unmanaged](#unmanaged-file) or [excluded files](#excluded-file) and [authorized](#authorization) compatibility exceptions. This does not mean Minecraft has been run successfully, or that every [successful command](#success) certifies the [installation](#installation) as satisfied.

### Drift

A difference between [managed files](#managed-file) and the [artifact](#artifact) content expected by the applicable [locked artifact](#locked-artifact), such as a missing or modified [file](#file). [Repair](#repair) corrects the [file](#file) state; it does not silently [adopt](#adoption) the changed content as a new [locked artifact](#locked-artifact).

## Ownership

### Managed file

A [file](#file) within MMM's declared and [resolved](#resolution) management relationship, including a previously managed file awaiting [removal](#removal) during [reconciliation](#reconciliation). Management is not inferred from directory presence or successful [recognition](#recognition) alone. A managed file can be missing or mismatched; [exclusion protection](#excluded-file) still applies to affected paths.

### Unmanaged file

A [local file](#file) outside MMM's management relationship. [Visible](#visible-file) unmanaged jars are legitimate and remain untouched by ordinary managed operations. An unmanaged file may still be [recognized](#recognition) as a known [platform artifact](#artifact).

### Excluded file

A [file](#file) protected from [scanning](#discovery), [unmanaged reporting](#unmanaged-file), [adoption](#adoption), [updates](#update), [removal](#removal) and [pruning](#pruning) by `.mmmignore` or the baseline `.disabled` rule. Exclusion is not evidence of [installation satisfaction](#satisfied-installation) and does not permit overwriting a destination [collision](#collision), even with [force](#force).

### Ignored file

A [file](#file) matched by `.mmmignore`, with patterns relative to the [mods directory](#mods-directory). Use “[excluded file](#excluded-file)” when also including the separate `.disabled` rule. Invalid ignore patterns stop mutation rather than silently removing protection.

### Disabled file

In the baseline, a [file](#file) whose name ends in `.disabled`, receiving [exclusion](#excluded-file) treatment. This is not first-class [managed enabled/disabled state](#managed-enabled-and-disabled-state) and does not satisfy a declared enabled counterpart.

### Visible file

A [file](#file) not hidden from MMM by [exclusion rules](#excluded-file) and within the relevant [operation](#operation)'s [discovery](#discovery) scope. “Visible” does not mean [managed](#managed-file), [recognized](#recognition) or currently displayed on screen.

### Discovery

Finding candidate [local jars](#file) within the scan boundary. Discovery neither establishes [platform identity](#mod-identity) nor grants management [authority](#authorization).

### Recognition

Identifying a [discovered](#discovery) [file](#file) as a [platform](#platform) [artifact](#artifact). Scan distinguishes known matches, unknown results and uncertain results caused by unresolved [lookup](#lookup) [failures](#failure). Recognition alone does not authorize [adoption](#adoption).

### Adoption

Explicitly accepting a [recognized](#recognition) local [artifact](#artifact) into the [modlist](#modlist) and [lockfile](#lockfile), including accepting changed content as a new [locked artifact](#locked-artifact) when [authorized](#authorization). Adoption records the [discovered](#discovery) [artifact](#artifact); it does not download a newer version.

### Repair

Restoring a missing or mismatched [managed file](#managed-file) to its expected [artifact](#artifact) during [installation](#install-operation). Repair preserves the applicable [locked artifact](#locked-artifact) rather than accepting modified local content as its replacement.

### Reconciliation

Bringing the [modlist](#modlist), [lockfile](#lockfile) and [managed files](#managed-file) into agreement under the [operation](#operation)'s [lookup](#lookup) policy. It can [resolve](#resolution) additions or changed [constraints](#constraint), [repair](#repair) files and remove previously managed [artifacts](#artifact) no longer declared. It is not automatic [adoption](#adoption) or necessarily an upgrade.

### Collision

A conflict at an [operation](#operation)'s destination with an unrelated or [protected file](#excluded-file). It fails the affected [operation](#operation) instead of authorizing silent replacement; independent work can continue when safe.

## Operations and outcomes

### Initialization

Creating a [modlist](#modlist) and an empty [lockfile](#lockfile) through `init` or the shared setup flow. It does not download or [adopt](#adoption) [mods](#mod). Resetting existing metadata requires the documented reset [authority](#authorization).

### Addition

Adding or updating a [mod config](#mod-config) for an explicitly selected [project](#project), applying requested [lookup](#lookup) settings and [reconciling](#reconciliation) its [installation](#installation) through `add`. An addition that saves only the [mod config](#mod-config) under [force](#force) remains an [incomplete installation outcome](#incomplete-execution).

### Install operation

The `install` [command](#command)'s [reconciliation](#reconciliation) of the managed [installation](#installation) while preserving [valid locked artifacts](#valid-locked-artifact). It [repairs](#repair) [drift](#drift) and [resolves](#resolution) changed [mod configs](#mod-config) without opportunistically upgrading [valid locked artifacts](#valid-locked-artifact). Use this term to distinguish the [operation](#operation) from the [installation](#installation) it operates on.

### Update

[Reconciliation](#reconciliation) with a [lookup](#lookup) policy seeking newer [eligible](#eligibility) [artifacts](#artifact) for [unpinned](#pin) [mods](#mod). An update requires later publication and different content; [pins](#pin) remain binding. It does not require first installing the old [locked artifact](#locked-artifact).

### Removal

Removing selected [mod configs](#mod-config), [lock entries](#lock-entry) and managed [artifacts](#artifact) through `remove`. An unsuccessful [file](#file) deletion is not a completed removal. This differs from [pruning](#pruning) [unmanaged jars](#unmanaged-file).

### Inspection

Reporting what can be established without changing [installation files](#file), [modlist](#modlist) or [lockfile](#lockfile). `list` and `test` own inspection work. An explicitly accepted [setup](#initialization) or correction is a separate flow, not an inspection mutation.

### Compatibility check

The `test` [operation](#operation)'s assessment of [platform](#platform)-declared [eligibility](#eligibility) for a [Minecraft target](#minecraft-target). It distinguishes [compatible](#compatible), [incompatible](#incompatible) and [inconclusive results](#inconclusive-result); it does not launch Minecraft.

### Compatible

A conclusive positive [platform](#platform)-declared [eligibility](#eligibility) finding for the checked [target](#minecraft-target) and applicable [constraints](#constraint). It does not establish that Minecraft can successfully run the [mod](#mod).

### Incompatible

A conclusive negative [platform](#platform)-based [compatibility finding](#compatibility-check). A failed request or unavailable service is not evidence of incompatibility. Use “runtime failure” for an observed Minecraft execution problem, which MMM does not diagnose in the baseline.

### Inconclusive result

A result where available evidence cannot establish the requested finding, such as a compatibility [lookup](#lookup) that fails. A [compatibility run](#compatibility-check) containing both [incompatibilities](#incompatible) and inconclusive checks is [incomplete overall](#incomplete-execution). Scan uses “uncertain” for unresolved [recognition](#recognition) results.

### Version change

The `change` [operation](#operation) that changes the common [Minecraft target](#minecraft-target) and prepares its corresponding managed [installation](#installation). It has coordinated preparation and switching, unlike independent per-mod [reconciliation](#reconciliation). A request for the already-configured [target](#minecraft-target) is a [no-op](#no-op), not a [repair](#repair) request.

### Retained artifact

An existing [artifact](#artifact) kept during forced [version change](#version-change) when an [eligible](#eligibility) [target](#minecraft-target) replacement is unavailable. Explicit retention authorizes that [artifact](#artifact) for that [target](#minecraft-target); later [installation](#install-operation) preserves or reproduces it. Retention neither disables the [mod](#mod) nor proves runtime compatibility.

### Pruning

Explicit deletion of [visible](#visible-file) [unmanaged](#unmanaged-file) `.jar` [files](#file) through `prune`, with [confirmation](#confirmation) or [force](#force). It excludes [managed](#managed-file) and [protected files](#excluded-file), does not recurse, and refuses deletion without [lockfile](#lockfile) [evidence](#resolution-evidence).

### Success

Satisfaction of the requested [command](#command) outcome, including an already-satisfied [no-op](#no-op). Success is [command](#command)-specific: successful [same-target change](#version-change) does not certify that missing [mod](#mod) [files](#file) were [repaired](#repair). A completed check finding [incompatibility](#incompatible) has a distinct non-success outcome.

### Partial success

Completion of some independent work while other requested work [fails](#failure) or remains unresolved. Completed work remains consistent, but the overall run is not [successful](#success).

### Incomplete execution

A run that cannot establish completion of the requested outcome, including [partial failure](#partial-success), unresolved [installation](#installation) or [inconclusive checks](#inconclusive-result). Reports retain completed results and identify remaining work.

### Failure

An unsuccessful [operation](#operation) or required step, such as a request, download, integrity check or metadata write. State the affected item and known reason; failure does not imply that all preceding work was undone.

### Invalid invocation

A request that cannot be accepted as a valid [command invocation](#invocation). It is distinct from execution [failure](#failure) or interruption. Supported field values can enter the documented correction flow; malformed [command](#command) syntax does not become valid through that mechanism.

### No-op

A [successful](#success) [operation](#operation) requiring no material changes for its command-specific outcome. It is not a [failure](#failure) or a guarantee that every possible [installation](#installation) [operation](#operation) would also do nothing.

### Retry

Another attempt after unfinished or [failed work](#failure). Retrying converges on the requested state without duplicates while retaining completed independent work. Automatic request and cleanup retries are bounded; retry safety does not freeze upstream releases or promise identical logs.

### Recovery

Work that restores or establishes consistency after [failure](#failure) or interruption, with the actual remaining state reported if recovery is incomplete. Qualify “[setup recovery](#initialization)” or “input correction” when referring to shared interactive remediation instead. Recovery does not promise that nothing changed.

### Safe cancellation

Stopping new work and unfinished downloads while retaining completed independent changes and finishing necessary consistency or [recovery](#recovery) work. The first `Ctrl+C` requests it. Required [recovery](#recovery) has no automatic shutdown timeout.

### Forced termination

The emergency exit requested by a second interruption after the warning. It may leave [recovery](#recovery) unfinished and is outside the [safe-cancellation guarantee](#safe-cancellation). It is distinct from a [command](#command)'s `--force` option.

## Execution and presentation

### Invocation

One execution of the MMM process that completes and returns control to the shell. It can include shared [setup](#initialization) or correction before resuming the requested [command](#command).

### Command

A named CLI capability requested by the [operator](#operator), such as `install` or `list`. Distinguish the command interface from its [flow](#command-flow), [reusable capabilities](#capability-component) and underlying [business operations](#business-operation).

### Operation

A unit of requested work or its execution. Qualify its scope when material: [command](#command) operation, per-mod operation, [file](#file) operation or [business operation](#business-operation). Do not assume every operation is a separate process or an all-or-nothing transaction.

### Interactive execution

Execution with terminal input and output where prompting is allowed and [interactive controls](#visual-primitive) can collect missing decisions. Supplying `--unattended` prohibits prompting even in a terminal.

### Unattended execution

Execution with `--unattended`: never ask questions; use supplied policies and documented defaults or fail when required input or [authority](#authorization) is missing. Terminal progress can still render dynamically. It does not imply [force](#force).

### Redirected execution

Execution where input or output is redirected. It never prompts and emits plain append-only output without animation or cursor-control sequences. It grants no additional [authority](#authorization). “Non-interactive” can describe the absence of interaction, but is not a separate MMM mode flag.

### Authorization

The [operator](#operator)'s permission for a particular action, expressed through the [command](#command)'s documented inputs, flags or accepted choices. Prompt availability and authority are separate: [unattended execution](#unattended-execution) does not supply missing permission.

### Confirmation

An explicit decision accepting a described action. Its scope is the action presented: [init](#initialization)'s metadata-reset confirmation differs from its final [setup](#initialization) confirmation. Declining or cancelling is not consent.

### Force

[Command](#command)-specific [authorization](#authorization) supplied through `--force`. Its meaning belongs to each [command](#command); it is neither a universal “yes” nor permission to bypass integrity, [exclusions](#excluded-file) or [recovery](#recovery).

### Active display

The mutable presentation of pending work, progress and [controls](#visual-primitive). It can repaint and reorganize while the [permanent transcript](#permanent-transcript) retains [settled history](#permanent-transcript). It is not synonymous with an alternate screen.

### Permanent transcript

The durable record of [settled results](#settled-result) and relevant answered decisions. Results [commit](#transcript-commit) once in completion order and survive repaints and exit with pre-command shell history. “Permanent” describes terminal-history lifetime, not a promised log [file](#file) or dedicated viewer.

### Settled result

An outcome ready to be [committed](#transcript-commit) once to the [permanent transcript](#permanent-transcript). A later correction is a new explicit event, not a rewrite. A settled per-item result does not mean the whole [command](#command) has completed.

### Transcript commit

The [terminal session](#terminal-session)'s act of adding a durable result or decision to the [transcript](#permanent-transcript). It is distinct from repainting [active content](#active-display) and from a Git commit.

### Active end

The current end of the [transcript](#permanent-transcript) where following new output resumes. Scrolling away preserves the [operator](#operator)'s reading position; returning shows current work and any pending prompt.

## Architecture and operational concepts

### Terminal session

The architectural owner of input routing, focus, scrolling, [active rendering](#active-display), [transcript commits](#transcript-commit) and terminal restoration. It coordinates presentation; [child components](#capability-component) do not independently take control of the terminal. This is a responsibility boundary, not a prescribed package, process or framework type.

### Command flow

The sequence and composition of [capabilities](#capability-component) for a requested [command](#command), including shared [recovery](#recovery) and return to the original [operation](#operation). It supplies domain data and decisions to [components](#capability-component); [command](#command)-specific orchestration can remain in [command](#command) packages.

### Capability component

A reusable interaction owned end to end, including its state, behaviour, interaction and output contract. Examples include [confirmation](#confirmation), choosing an option and [initialization](#initialization). Sharing [styles](#visual-primitive) alone does not establish a shared capability; components report choices and outcomes through explicit contracts.

### Visual primitive

A presentation building block such as a control, style or icon. It supports consistent [capabilities](#capability-component) but does not own the complete [command](#command) interaction or [business operation](#business-operation).

### Business operation

Rendering-independent work that [resolves](#resolution) [artifacts](#artifact) or performs filesystem and metadata changes through explicit boundaries. [Interactive](#interactive-execution) and [unattended flows](#unattended-execution) invoke the same [operations](#operation) and consume their outcomes. It does not acquire separate [lookup](#lookup) or safety rules from its presentation.

### Minecraft version manifest

Mojang's metadata identifying Minecraft versions, used to validate [explicit targets](#minecraft-target) and determine the [latest stable release](#latest). It is not the [modlist](#modlist) or [lockfile](#lockfile).

### Manifest cache

A locally persisted, successfully validated [Minecraft version manifest](#minecraft-version-manifest) with its fetch time. When refresh fails, a valid [cached manifest](#minecraft-version-manifest) can be used regardless of age with the required warning. Absence of a version from stale evidence does not establish invalidity. This is not a promise of full offline operation.

### Diagnostics

Additional information helping an [operator](#operator) explain a [failure](#failure). `--debug` requests diagnostic detail. Diagnostics are distinct from ordinary results, required metadata writes and [telemetry](#telemetry).

### Performance recording

A local performance artifact explicitly requested with `--perf`, with location controlled by `--perf-out-dir`. Export [failure](#failure) does not turn an otherwise successful [mod](#mod) [operation](#operation) into [failure](#failure). It is separate from permitted [telemetry](#telemetry) metrics.

### Telemetry

The explicitly permitted operational metrics and documented stable machine identifier collected by default, with `MMM_DISABLE_TELEMETRY` as the opt-out. It excludes personal information, credentials, [mod identities](#mod-identity), private paths, arbitrary arguments and raw errors. Do not describe it as unqualified anonymity or treat [diagnostics](#diagnostics) as implicitly authorized telemetry.

## Future concepts

These concepts describe future capabilities outside the Go-port baseline.

### Guided constraint resolution

A future interaction explaining which [constraints](#constraint) exclude available [artifacts](#artifact) and offering explicit per-mod changes. Baseline search correction edits [platform](#platform) and [ID](#project-id); it is not this richer capability.

### Dependency management

Future automatic [resolution](#resolution) and [installation](#install-operation) of [mods](#mod) required by other [mods](#mod), with additional [lookup](#lookup), ordering and [failure](#failure) semantics still to be defined. Baseline [operators](#operator) explicitly include a [mod config](#mod-config) for each [mod](#mod); MMM does not silently introduce dependencies.

### Managed enabled and disabled state

Future first-class state for managed [artifacts](#artifact), preserved through [installation](#install-operation) and [updates](#update) and exposed through [listing](#inspection) and enable/disable operations. It must recognize external toggles. This is distinct from baseline `.disabled` [exclusion](#excluded-file) and from `.mmmignore`.

## Distinctions in practice

| Situation | Precise description |
| --- | --- |
| A [pin](#pin) change [resolves](#resolution) B but downloading B [fails](#failure); [jar](#file) A remains. | The [mod config](#mod-config) requests B, [installation](#installation) still contains A, and [recovery](#recovery) evidence is retained. A's presence does not satisfy B. [Resolution](#resolution) success is not download success. |
| The [lockfile](#lockfile) selects A but its [local file](#file) is missing. | A is [locked](#locked-artifact) but not present. [Installation](#install-operation) reproduces A when available; it does not substitute a different [artifact](#artifact) silently. |
| Forced [version change](#version-change) [retains](#retained-artifact) A despite missing [target](#minecraft-target) compatibility. | A is a [valid locked artifact](#valid-locked-artifact) [authorized](#authorization) for that [target](#minecraft-target). Its platform compatibility limitation remains reportable; runtime success is unproven. |
| `example.jar` is renamed to `example.jar.disabled`. | The renamed [file](#file) is [excluded](#excluded-file). It does not satisfy the declared enabled counterpart; baseline [installation](#install-operation) may recreate `example.jar` alongside it. |
