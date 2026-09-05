# Legacy terminal test removal ledger

This ledger records terminal-related tests and helpers removed while replacing the custom terminal harness with tui-test.

> [!WARNING]
> This is historical evidence, not an approved product contract. An entry describes what a test asserted, not what MMM should do. Every entry remains `Unassessed` until the product requirement is reviewed independently.

## Recovery and baseline

The inventory was taken from commit `de8d16e886810bd200bd70b7d921cb8083e9c66c` before removal.

Recover any source or snapshot with:

```shell
git show de8d16e886810bd200bd70b7d921cb8083e9c66c:<original-path>
```

After the removal is committed, the equivalent form is `git show <removal-commit>^:<original-path>`.

Unless an entry says otherwise, legacy terminal tests used `MMM_TEST`, deterministic ASCII colour and Unicode capability fixtures, blocked live HTTP, normalized terminal output, and a two-second custom polling wait. These controls were implementation details of the deleted harness.

## Reconciliation

| Removed surface | Total |
| --- | ---: |
| Top-level tests | 151 |
| Harness self-tests | 87 |
| Command PTY tests | 42 |
| Unmanaged-notice snapshot tests | 12 |
| Lock-sync snapshot tests | 2 |
| Tests removed from mixed files | 8 |
| Whole files | 45 |
| Harness implementation files | 13 |
| Legacy test source files | 13 |
| Harness test source files | 7 |
| Snapshot files | 12 |
| Snapshot entries | 64 |

At top-level-test granularity, 58 tests asserted documented behaviour through implementation-coupled expectations and 93 asserted harness or rendering mechanisms only. Twelve unmanaged-notice tests were also duplicates across commands. Static inspection identified no expectation as definitely contradictory; that is not an endorsement of correctness.

Every item below has the future disposition `Unassessed` and the removal reason `legacy expectation coupled to the deleted custom terminal harness` unless stated otherwise. When an item is assessed, append a note under the applicable ledger table in the form `Assessment: <ID> — <decision>; requirement: <link>; replacement scenario: <link or None>`. Keep the historical row unchanged and never relabel it as approved behavior.

## Command PTY suites

The historical classifications used the per-command flow documents, the command specifications available at the recorded baseline, and the shared [interaction guidelines](../interactions/interaction-guidelines.md) as documentary sources. Reassess the entries against current [product intent](../intent.md) and [command guides](../commands/README.md); historical classifications do not establish current authority. Classification is `documented, implementation-coupled` unless stated otherwise. Fixtures were inline, in an afero memory filesystem, or in a test-owned temporary directory; there are no external fixture files.

### Change

| ID | Original test | Inputs and environment | Historical observation | Asset |
| --- | --- | --- | --- | --- |
| LT-PTY-CHANGE-001 | `cmd/mmm/change/pty_integration_test.go::TestChangeCommandInteractivePTYOutput` | 120×40; `change 1.21.1`; inline Alpha/Modrinth and Beta/CurseForge; Beta supported with a 512 KiB of 1 MiB download | Expected header, unmanaged notice, compatibility and downloading sections; expected switching section absent | None |

### Install

All three tests used 30 inline Modrinth mods and no key input.

| ID | Original test | Dimensions and state | Historical observation | Snapshot entry |
| --- | --- | --- | --- | --- |
| LT-PTY-INSTALL-001 | `TestInstallCommandInteractivePTYFinalSnapshotLongListMediumHeight` | 120×25; all items completed successfully | Full final success list, header and summary | Same as test name |
| LT-PTY-INSTALL-002 | `TestInstallCommandInteractivePTYFinalSnapshotLongListTallHeight` | 80×40 on Windows, 80×80 elsewhere; all completed | Full final list remained visible | Same as test name |
| LT-PTY-INSTALL-003 | `TestInstallCommandInteractivePTYRunningSnapshotLongListShortHeight` | 120×12; execution held after success messages | Running render retained every success item and header | Same as test name |

Source: `cmd/mmm/install/pty_integration_test.go`. Asset: `cmd/mmm/install/__snapshots__/pty_integration_test.snap`.

### Remove

| ID | Original test | Inputs and actions | Historical observation | Asset |
| --- | --- | --- | --- | --- |
| LT-PTY-REMOVE-001 | `TestRemoveCommandInteractivePTYIncludesSummary` | 120×40; temporary config, lock and `mod-a.jar`; `--force mod-a` | Successful summary | None |
| LT-PTY-REMOVE-002 | `TestRemoveCommandInteractivePTYConfirmSnapshotShortHeight` | 120×25; Mod A and B config, lock and jars; Enter accepted default No | Prompt, help, collapsed answer and cancellation | Snapshot entry |
| LT-PTY-REMOVE-003 | `TestRemoveCommandInteractivePTYConfirmSnapshotTallHeight` | 120×40 Windows, 120×80 elsewhere; same fixture and action | Tall prompt and cancellation frame | Snapshot entry |
| LT-PTY-REMOVE-004 | `TestRemoveCommandInteractivePTYSuccessSnapshotShortHeight` | 120×25; localized Yes token then Enter | Collapsed prompt, Mod A result and success summary | Snapshot entry |
| LT-PTY-REMOVE-005 | `TestRemoveCommandInteractivePTYSuccessSnapshotTallHeight` | 120×40 Windows, 120×80 elsewhere; localized Yes then Enter | Tall successful transcript | Snapshot entry |
| LT-PTY-REMOVE-006 | `TestRemoveCommandInteractivePTYSuccessLongListShowsHeaderWhenShort` | 120×12; 30 temporary mods; `--force mod-*` | Final result header remained visible | None |

Source: `cmd/mmm/remove/pty_integration_test.go`. Snapshot asset: `cmd/mmm/remove/__snapshots__/pty_integration_test.snap`. Prompt cases additionally set `LANG=en_GB.UTF-8`.

### Scan

| ID | Original test | Dimensions, fixture and action | Historical observation |
| --- | --- | --- | --- |
| LT-PTY-SCAN-001 | `TestScanCommandInteractivePTYRunningSnapshotShortHeight` | 120×25; 6 pending, 7 recognized, 6 unknown and 6 unsure; runner held | Running header, pending and recognized ordering |
| LT-PTY-SCAN-002 | `TestScanCommandInteractivePTYRunningSnapshotShortHeightManyMods` | 120×25; 30 realistic jar names; runner held | All sorted running rows and header |
| LT-PTY-SCAN-003 | `TestScanCommandInteractivePTYRunningSnapshotMediumHeight` | 120×12; categorized fixture; runner held | Full categorized view and repeated sticky header |
| LT-PTY-SCAN-004 | `TestScanCommandInteractivePTYFinalTranscriptShortHeight` | 120×25; categorized fixture completed | Recognized, unknown and unsure result transcript |
| LT-PTY-SCAN-005 | `TestScanCommandInteractivePTYRunningSnapshotTallHeight` | 80×40 Windows, 80×80 elsewhere; runner held | Full running sections |
| LT-PTY-SCAN-006 | `TestScanCommandInteractivePTYFinalTranscriptMediumHeight` | 120×12; categorized fixture completed | Medium final transcript |
| LT-PTY-SCAN-007 | `TestScanCommandInteractivePTYFinalTranscriptTallHeight` | 80×40 Windows, 80×80 elsewhere; categorized fixture completed | Tall final transcript |
| LT-PTY-SCAN-008 | `TestScanCommandInteractivePTYPromptConfirmAddedOnly` | 120×12; one scan match; localized Yes then Enter | Only Added section and Alpha, without stale prompt text |
| LT-PTY-SCAN-009 | `TestScanCommandInteractivePTYPromptDeclineCancelledOnly` | 120×12; one scan match; localized No then Enter | Only adoption-cancelled line |

Source and inline support: `cmd/mmm/scan/pty_integration_test.go`. All nine entries used `cmd/mmm/scan/__snapshots__/pty_integration_test.snap`. Prompt cases used `LANG=en_GB.UTF-8` and an afero configuration and lock.

### Test

| ID | Original test | Dimensions, fixture and action | Historical observation |
| --- | --- | --- | --- |
| LT-PTY-TEST-001 | `TestTestCommandInteractivePTYOutput` | 120×40; target 1.21.1; Alpha compatible, Beta inconclusive | Header, compatible and inconclusive sections, summary and hint |
| LT-PTY-TEST-002 | `TestTestCommandInteractivePTYRunningShowsHeadersWhenShort` | 120×25; Alpha compatible, Beta incompatible, Gamma checking; runner held | Both running section headers and items |
| LT-PTY-TEST-003 | `TestTestCommandInteractivePTYRunningSnapshotMediumHeight` | 120×12; Alpha compatible, Beta and Gamma checking | Compatibility running frame |
| LT-PTY-TEST-004 | `TestTestCommandInteractivePTYRunningScrollsToNotCompatible` | 60×6; Alpha–Epsilon compatible, Zeta incompatible; PageDown bytes | Zeta became visible after keyboard scrolling |
| LT-PTY-TEST-005 | `TestTestCommandInteractivePTYRunningSnapshotTallHeightFailure` | 80×40 Windows, 80×80 elsewhere; mixed compatible, incompatible and checking | Both sections in tall frame |
| LT-PTY-TEST-006 | `TestTestCommandInteractivePTYRunningMouseScrollsToNotCompatible` | 60×25; Alpha–Epsilon compatible, Zeta incompatible; repeated encoded wheel-down input | Zeta became visible after mouse scrolling |
| LT-PTY-TEST-007 | `TestTestCommandInteractivePTYFinalTranscriptSuccessShortHeight` | 120×6; three compatible mods completed | Full final transcript and success summary |
| LT-PTY-TEST-008 | `TestTestCommandInteractivePTYFinalTranscriptIncludesSummaryAfterScroll` | 60×6; five compatible and Zeta incompatible; wheel input before completion | Complete final transcript and unsupported summary |
| LT-PTY-TEST-009 | `TestTestCommandInteractivePTYFinalTranscriptInconclusiveMediumHeight` | 120×12; Alpha and Gamma compatible, Beta timed out | Inconclusive section, summary and hint |
| LT-PTY-TEST-010 | `TestTestCommandInteractivePTYFinalTranscriptIncludesSummaryWithoutScroll` | 60×6; five compatible and Zeta incompatible; no scrolling | Complete final transcript and unsupported summary |

Source and inline support: `cmd/mmm/test/pty_integration_test.go`. Asset: `cmd/mmm/test/__snapshots__/pty_integration_test.snap`.

### Update

The standard fixture contained 12 mods: ten update candidates including Gamma at 20% download, Pinned skipped, and Failed with `cmd.update.error.platform`. The long fixture contained 26 mods: 24 update candidates plus Pinned and Failed. Finalization alternated updated and up-to-date results.

| ID | Original test | Dimensions and state | Historical observation | Asset |
| --- | --- | --- | --- | --- |
| LT-PTY-UPDATE-001 | `TestUpdateCommandInteractivePTYRunningSnapshotShortHeight` | 120×25; standard runner held | Running header, sections and items | Snapshot entry |
| LT-PTY-UPDATE-002 | `TestUpdateCommandInteractivePTYRunningSnapshotMediumHeight` | 120×12; standard runner held | Medium running frame | Snapshot entry |
| LT-PTY-UPDATE-003 | `TestUpdateCommandInteractivePTYRunningSnapshotTallHeight` | 80×40 Windows, 80×80 elsewhere; standard runner held | Tall running frame | Snapshot entry |
| LT-PTY-UPDATE-004 | `TestUpdateCommandInteractivePTYFinalSnapshotShortHeight` | 120×25; standard completed | Final sections and incomplete summary/hint | Snapshot entry |
| LT-PTY-UPDATE-005 | `TestUpdateCommandInteractivePTYFinalSnapshotMediumHeight` | 120×12; standard completed | Medium final transcript | Snapshot entry |
| LT-PTY-UPDATE-006 | `TestUpdateCommandInteractivePTYFinalSnapshotTallHeight` | 80×40 Windows, 80×80 elsewhere; standard completed | Tall final transcript | Snapshot entry |
| LT-PTY-UPDATE-007 | `TestUpdateCommandInteractivePTYFinalSnapshotLongListShortHeight` | 120×25; long completed | Long final transcript | Snapshot entry |
| LT-PTY-UPDATE-008 | `TestUpdateCommandInteractivePTYFinalSnapshotLongListMediumHeight` | 120×12; long completed | Long medium transcript | Snapshot entry |
| LT-PTY-UPDATE-009 | `TestUpdateCommandInteractivePTYFinalSnapshotLongListTallHeight` | 80×40 Windows, 80×80 elsewhere; long completed | Long tall transcript | Snapshot entry |
| LT-PTY-UPDATE-010 | `TestExtractPTYTranscriptSnapshotReportsMissingTranscript` | No terminal; malformed transcript input | Custom transcript extractor returned an error | None; classification `implementation-coupled mechanism` |
| LT-PTY-UPDATE-011 | `TestUpdateCommandInteractivePTYRunningSnapshotLongListShortHeight` | 120×25; long runner held | Long running state and failure markers | Snapshot entry |
| LT-PTY-UPDATE-012 | `TestUpdateCommandInteractivePTYRunningSnapshotLongListMediumHeight` | 120×12; long runner held | Long medium running state | Snapshot entry |
| LT-PTY-UPDATE-013 | `TestUpdateCommandInteractivePTYRunningSnapshotLongListTallHeight` | 80×40 Windows, 80×80 elsewhere; long runner held | Long tall running state | Snapshot entry |

Source and inline support: `cmd/mmm/update/pty_integration_test.go`. Asset: `cmd/mmm/update/__snapshots__/pty_integration_test.snap`, except LT-PTY-UPDATE-010.

## Shared rendering suites

### Unmanaged-file notice

Each command had a short-row and tall-row snapshot. The tests created an in-memory configuration, lock and `unmanaged.jar`, ran an unattended command, expected a handled unmanaged-files error, and captured normalized stdout and stderr. Change and test also expected exit code 1. The historical output used the `cmd.list.unmanaged.header`, `description` and `cta` keys plus an error icon and filename.

| IDs | Original source | Tests | Asset |
| --- | --- | --- | --- |
| LT-UNMANAGED-ADD-SHORT/TALL | `cmd/mmm/add/unmanaged_notice_snapshot_test.go` | `TestAddUnmanagedNoticeSnapshotShortHeight`, `TestAddUnmanagedNoticeSnapshotTallHeight` | `cmd/mmm/add/__snapshots__/unmanaged_notice_snapshot_test.snap` |
| LT-UNMANAGED-CHANGE-SHORT/TALL | `cmd/mmm/change/unmanaged_notice_snapshot_test.go` | `TestChangeUnmanagedNoticeSnapshotShortHeight`, `TestChangeUnmanagedNoticeSnapshotTallHeight` | `cmd/mmm/change/__snapshots__/unmanaged_notice_snapshot_test.snap` |
| LT-UNMANAGED-INSTALL-SHORT/TALL | `cmd/mmm/install/unmanaged_notice_snapshot_test.go` | `TestInstallUnmanagedNoticeSnapshotShortHeight`, `TestInstallUnmanagedNoticeSnapshotTallHeight` | `cmd/mmm/install/__snapshots__/unmanaged_notice_snapshot_test.snap` |
| LT-UNMANAGED-LIST-SHORT/TALL | `cmd/mmm/list/unmanaged_notice_snapshot_test.go` | `TestListUnmanagedNoticeSnapshotShortHeight`, `TestListUnmanagedNoticeSnapshotTallHeight` | `cmd/mmm/list/__snapshots__/unmanaged_notice_snapshot_test.snap` |
| LT-UNMANAGED-REMOVE-SHORT/TALL | `cmd/mmm/remove/unmanaged_notice_snapshot_test.go` | `TestRemoveUnmanagedNoticeSnapshotShortHeight`, `TestRemoveUnmanagedNoticeSnapshotTallHeight` | `cmd/mmm/remove/__snapshots__/unmanaged_notice_snapshot_test.snap` |
| LT-UNMANAGED-TEST-SHORT/TALL | `cmd/mmm/test/unmanaged_notice_snapshot_test.go` | `TestTestUnmanagedNoticeSnapshotShortHeight`, `TestTestUnmanagedNoticeSnapshotTallHeight` | `cmd/mmm/test/__snapshots__/unmanaged_notice_snapshot_test.snap` |

Short row limit was 25; tall was 40 on Windows and 80 elsewhere. Classification: `documented, duplicated, implementation-coupled`. Authority: the unmanaged notice section of the [interaction guidelines](../interactions/interaction-guidelines.md) and each command flow/spec.

### Lock synchronization

| ID | Original test | Inputs and actions | Historical observation | Asset |
| --- | --- | --- | --- | --- |
| LT-LOCKSYNC-001 | `internal/locksync/output_snapshot_test.go::TestLockSyncPromptSnapshots` | 80×25 and 80×80; Alpha missing, Beta invalid path, Gamma present; select Add/Delete/Ignore/Skip then Enter | Initial prompt and four answered states at both sizes; 10 entries | `internal/locksync/__snapshots__/output_snapshot_test.snap` |
| LT-LOCKSYNC-002 | `internal/locksync/output_snapshot_test.go::TestLockSyncSummarySnapshots` | In-process install buffer with the same extras and four policies | Four policy summaries | Same asset; 4 entries |

Classification: `documented, implementation-coupled`. Authority: the lockfile-sync section of the interaction guidelines.

## Tests removed from mixed files

Only the named tests and their `testutil/terminal` imports were removed. Other tests in these files remain.

| ID | Original test | Historical setup and observation | Classification |
| --- | --- | --- | --- |
| LT-MIX-REMOVE-001 | `cmd/mmm/remove/remove_test.go::TestRunRemoveNonTTYRequiresForce` | Fake non-TTY devices, English rendered text, memory config/lock/jar; expected refusal, `--force` guidance and unchanged jar | Documented, implementation-coupled |
| LT-MIX-REMOVE-002 | `cmd/mmm/remove/remove_test.go::TestRunRemoveCancelDoesNotCreateLockFile` | Fake TTY and declined confirmation; expected zero removals, interactive mode, no lock creation and unchanged config | Documented, implementation-coupled |
| LT-MIX-REMOVE-003 | `cmd/mmm/remove/remove_test.go::TestRunRemoveNonTTYRefusalDoesNotCreateLockFile` | Fake non-TTY; expected refusal, no lock creation and unchanged config | Documented, implementation-coupled |
| LT-MIX-SCAN-001 | `cmd/mmm/scan/run_helpers_test.go::TestRunInteractiveScanSetsWindowSize` | Fake terminal size 120×33; expected dimensions copied into the model | Implementation-coupled mechanism |
| LT-MIX-SCAN-002 | `cmd/mmm/scan/model_test.go::TestScanRenderWithStickyHeaderSkipsWhenWindowHeightZero` | Height 0 and normalized whitespace; expected duplicated content/header rendering | Implementation-coupled rendering |
| LT-MIX-SCAN-003 | `cmd/mmm/scan/model_test.go::TestScanRenderWithStickyHeaderEchoesWhenContentExceedsWindow` | Height 2 and overflowing body; expected repeated header | Implementation-coupled rendering |
| LT-MIX-UPDATE-001 | `cmd/mmm/update/model_test.go::TestUpdateRenderWithStickyHeaderSkipsWhenWindowHeightZero` | Height 0; expected content plus repeated header | Implementation-coupled rendering |
| LT-MIX-UPDATE-002 | `cmd/mmm/update/model_test.go::TestUpdateRenderWithStickyHeaderEchoesWhenContentExceedsWindow` | Height 2 and overflowing body; expected repeated header | Implementation-coupled rendering |

## Deleted harness

The deleted implementation consisted of 13 files and 159 top-level declarations. These groups collectively owned the terminal mechanism; none is a product requirement.

| ID | Original source | Removed responsibility |
| --- | --- | --- |
| LT-HARNESS-CORE-BUFFER | `testutil/terminal/buffer.go` | Thread-safe buffer and snapshot reader |
| LT-HARNESS-CORE-DETECT | `testutil/terminal/detection.go` | Overrides for terminal detection and file descriptors |
| LT-HARNESS-CORE-DEVICE | `testutil/terminal/device.go` | Fake terminal file/device with buffered I/O |
| LT-HARNESS-CORE-FIXTURE | `testutil/terminal/fixtures.go` | MMM_TEST, colour, Unicode, random and live-HTTP fixtures |
| LT-HARNESS-CORE-INPUT | `testutil/terminal/input.go` | Key bytes and SGR mouse encoding |
| LT-HARNESS-CORE-NORMALIZE | `testutil/terminal/normalize.go` | ANSI/OSC/CSI stripping and whitespace normalization |
| LT-HARNESS-CORE-TYPES | `testutil/terminal/types.go` | Terminal capabilities and dimensions |
| LT-HARNESS-PTY | `testutil/terminal/pty/harness.go` | Session setup, capture, resize, input, polling, timeout and cleanup |
| LT-HARNESS-PTY-UNIX | `testutil/terminal/pty/pty_backend_unix.go`, `normalize_read_error_unix.go` | Unix PTY allocation, resize and EIO normalization |
| LT-HARNESS-PTY-WINDOWS | `testutil/terminal/pty/pty_backend_windows.go`, `normalize_read_error_windows.go` | Windows console allocation, resize, pipe ownership and error normalization |
| LT-HARNESS-TEATEST | `testutil/terminal/teatest/harness.go` | In-process Bubble Tea session, input, wait and final-model helpers |

The six command PTY sources also contained 78 suite-local support declarations: install 11, remove 13, scan 15, test 22, update 17 and change 0. Recover them from the command source paths listed above.

For exact reconciliation, the deleted command-local helper groups were:

- Install: `runInstallPTYRunningSnapshotWithMods`, `runInstallPTYFinalSnapshotWithMods`, `normalizeInstallViewOutput`, `normalizeInstallFinalScreenOutput`, `normalizeInstallVisibleOutput`, `normalizeInstallRunningOutput`, `trimToLastInstallFrame`, `cursorHomeSequence`, `collapseInstallBlankLines`, `installTallSnapshotRows`, `sampleInstallModsLongList`.
- Remove: `runRemovePTYConfirmSnapshot`, `runRemovePTYSuccessSnapshot`, `writeRemoveFixture`, `writeRemoveLongListFixture`, `waitForRemoveOutput`, `normalizeRemovePromptSnapshot`, `normalizeRemoveSuccessSnapshot`, `extractRemoveSuccessSummary`, `trimToLastFrame`, `tallSnapshotRows`, `cursorHomeSequence`, `collapseDuplicateLines`, `reorderRemoveResultHeader`.
- Scan: `commandWithRunner`, `finalizePTYRun`, `waitForOutput`, `tallSnapshotRows`, `normalizePTYSnapshot`, `normalizePromptPTYSnapshot`, `trimAfterAltScreenExit`, `normalizeViewportSnapshot`, `trimBeforeSection`, `trimToLastFrame`, `normalizeScanVisibleOutput`, `scanItemsFromFileNames`, `lastSortedFileName`, `sampleModFileNames`, `cursorHomeSequence`.
- Test: `normalizeTestPTYOutput`, `normalizeTestLines`, `normalizeTestPTYOutputSection`, `preferredTestFrameMarkers`, `outputHasLine`, `dropCompatibilitySection`, `tallSnapshotRows`, `trimTestOutputToLastFrame`, `normalizedHasLine`, `containsTestSection`, `cursorFrameSequence`, `extractTestTranscript`, `lastLineIndexContaining`, `lastLineIndexWithPrefix`, `trimLeadingEmptyLines`, `trimToFirstHeaderBlock`, `trimToLastHeaderBlock`, `trimToSectionValue`, `trimToLastSectionBlock`, `adjustNormalizedForSectionMarker`, `preferredSectionMarker`, `finalizePTYRun`.
- Update: `runUpdatePTYRunningSnapshot`, `tallSnapshotRows`, `runUpdatePTYRunningSnapshotWithItems`, `normalizeUpdatePTYOutput`, `normalizeUpdateVisibleOutput`, `normalizeUpdatePTYRunningOutput`, `normalizeUpdateFailureLine`, `runUpdatePTYFinalSnapshot`, `runUpdatePTYFinalSnapshotWithItems`, `commandWithUpdateRunner`, `sampleUpdateItems`, `sampleUpdateItemsLongList`, `finalizeUpdateItems`, `updateIndexByConfig`, `extractPTYTranscriptSnapshot`, `summaryLineKey`, `countSummaryLines`.
- Change: no suite-local helper declarations.

Each unmanaged-notice source owned `run<Command>UnmanagedNoticeSnapshot` and `formatNoticeSnapshot`. Add also owned `noopDoer.Do`; add, change, install and list owned `tallSnapshotRows`, while remove and test shared the helper from their deleted PTY source. The lock-sync source owned `lockSyncSnapshotExtras` and `normalizeLockSyncSnapshot`.

### Harness self-tests

| ID | Original source | Removed tests |
| --- | --- | --- |
| LT-HARNESS-TEST-CORE | `testutil/terminal/terminal_test.go` | `TestBufferOperations`; `TestSnapshotReaderUpdatesBetweenReads`; `TestSnapshotReaderNilSource`; `TestSnapshotReaderEmptySource`; `TestSnapshotReaderPartialReads`; `TestDeviceAndTerminalDetection`; `TestApplyTerminalDetectionSingleDevice`; `TestApplyTerminalDetectionNoDevices`; `TestDeviceBasics`; `TestCapabilitiesHelpers`; `TestNormalizeOutput`; `TestEncodeSGRMouseSequence`; `TestSizeString`; `TestApplyFixturesReturnsDeterministicRand`; `TestApplyFixturesOverridesViewSupport`; `TestApplyFixturesNilTest`; `TestApplyFixturesDefault`; `TestApplyFixturesBlocksLiveHTTP` |
| LT-HARNESS-TEST-TEATEST | `testutil/terminal/teatest/harness_test.go` | `TestSessionCapturesOutputAndInputs`; `TestNewSessionNil`; `TestSessionFinalOutputAndModel`; `TestSessionNonTTYCapabilities`; `TestSessionWaitFinishedNoTimeout`; `TestSessionTimeoutWithoutHandler`; `TestSessionTimeoutHandler`; `TestSessionNilGuards` |
| LT-HARNESS-TEST-PTY-BASE | `testutil/terminal/pty/harness_test.go` | `TestSessionCapturesOutput` |
| LT-HARNESS-TEST-PTY-EXTRA | `testutil/terminal/pty/harness_additional_test.go` | `TestValidateSizeValid`; `TestValidateSizeInvalid`; `TestApplyPTYSizeNil`; `TestApplyPTYSizeInvalid`; `TestApplyPTYSizeSetError`; `TestNewSessionNil`; `TestNewSessionOpenError`; `TestNewSessionSizeError`; `TestStartCaptureNil`; `TestCloseFileIgnoresClosedFile`; `TestCloseFileReturnsOtherErrors`; `TestCloseFileTimesOutOnBlockedClose`; `TestCloseWithErrorIncludesCloseErrors`; `TestRegisterCleanupReportsCloseError`; `TestSessionNilGuards`; `TestSessionSlaveUsesOutputWhenSplit`; `TestSessionSlaveUsesInputWhenShared`; `TestSessionCloseIsIdempotent`; `TestSessionCloseWithoutReader`; `TestSessionCloseReportsExtraCloserError`; `TestCloseSetupReportsExtraCloserError`; `TestSessionCloseMissingReadError`; `TestSessionCloseWithoutReadErrorChannel`; `TestSessionCloseTimeout`; `TestSessionResizeInvalidSize`; `TestSessionNonTTYCapabilities`; `TestNormalizeReadError`; `TestWaitForIntervalOption`; `TestWaitForOutputAndClose`; `TestWaitForOutputAndCloseReportsCloseError` |
| LT-HARNESS-TEST-PTY-UNIX | `testutil/terminal/pty/pty_backend_unix_test.go` | `TestOpenPTYDefaultError`; `TestSetPTYSizeDefaultInvalidFile`; `TestSetPTYSizeDefaultSetError`; `TestWinsizeFromSizeInvalid`; `TestSetPTYSizeDefaultValid`; `TestSetPTYSizeDefaultInvalidSize` |
| LT-HARNESS-TEST-PTY-UNIX-READ | `testutil/terminal/pty/normalize_read_error_unix_test.go` | `TestNormalizeReadErrorEIO` |
| LT-HARNESS-TEST-PTY-WINDOWS | `testutil/terminal/pty/pty_backend_windows_test.go` | `TestVirtualPTYReadWriteAndClose`; `TestVirtualPTYNilReadWrite`; `TestVirtualPTYCloseReaderOnly`; `TestVirtualPTYCloseWriterOnly`; `TestVirtualPTYCloseBothSides`; `TestClosePipeFile`; `TestCloseWithJoin`; `TestConsoleRestoreClose`; `TestIsConsoleHandle`; `TestOpenPTYDefaultErrorPaths`; `TestSetPTYSizeDefaultBranches`; `TestSetPTYSizeDefaultExpandResize`; `TestSetPTYSizeDefaultMixedResize`; `TestSetPTYSizeDefaultMixedResizeColumnsExpand`; `TestSetPTYSizeDefaultExpandResizeBufferError`; `TestSetPTYSizeDefaultExpandResizeWindowError`; `TestSetPTYSizeDefaultMixedResizeWindowError`; `TestSetPTYSizeDefaultMixedResizeBufferError`; `TestSetPTYSizeDefaultMixedResizeFinalWindowError`; `TestSetPTYSizeDefaultErrors`; `TestWinConsoleHelpersErrorBranches`; `TestOpenConsoleFilesError`; `TestAppendConsoleClosers` |

## Deleted snapshot assets

This table is the asset reconciliation. Every deleted snapshot file appears once.

| Original asset | Entries |
| --- | ---: |
| `cmd/mmm/install/__snapshots__/pty_integration_test.snap` | 3 |
| `cmd/mmm/remove/__snapshots__/pty_integration_test.snap` | 4 |
| `cmd/mmm/scan/__snapshots__/pty_integration_test.snap` | 9 |
| `cmd/mmm/test/__snapshots__/pty_integration_test.snap` | 10 |
| `cmd/mmm/update/__snapshots__/pty_integration_test.snap` | 12 |
| `cmd/mmm/add/__snapshots__/unmanaged_notice_snapshot_test.snap` | 2 |
| `cmd/mmm/change/__snapshots__/unmanaged_notice_snapshot_test.snap` | 2 |
| `cmd/mmm/install/__snapshots__/unmanaged_notice_snapshot_test.snap` | 2 |
| `cmd/mmm/list/__snapshots__/unmanaged_notice_snapshot_test.snap` | 2 |
| `cmd/mmm/remove/__snapshots__/unmanaged_notice_snapshot_test.snap` | 2 |
| `cmd/mmm/test/__snapshots__/unmanaged_notice_snapshot_test.snap` | 2 |
| `internal/locksync/__snapshots__/output_snapshot_test.snap` | 14 |

## Explicit coverage gaps

Deletion removes historical checks for:

- interactive state, layout, scrolling, prompts and final transcripts for change, install, remove, scan, test and update;
- the shared unmanaged-files notice across six commands;
- lock-sync prompt and summary variants;
- non-TTY remove refusal and cancellation side effects;
- scan and update sticky-header rendering and scan terminal-size plumbing; and
- every implementation detail of the custom PTY, teatest, normalization, input and fixture system.

These gaps are deliberate. They must not be filled by copying the legacy expectation.

Immediately after removal, `make coverage` reported 99.9% statement coverage rather than the repository's 100% gate. The newly uncovered statements were in `cmd/mmm/change.runChange`, remove model/update/render/run paths, scan sticky-header and interactive-run paths, `cmd/mmm/test.runTest`, and `internal/locksync.lockSyncAnswerLabel`. This measurement is another locator for later requirement assessment, not authority to recreate the deleted assertions.

## Later assessment workflow

For each `Unassessed` entry:

1. Review the applicable product requirement independently of the deleted assertion.
2. Decide and document the intended behaviour.
3. If the decision requires product coverage, author a third-person Gherkin scenario using stable i18n keys and observable outcomes.
4. For that product scenario, implement new tui-test-backed actions and expectations.
5. Change production behaviour as needed until the approved scenario passes.
6. Update this ledger with the approved requirement and replacement scenario, or record `replacement scenario: None` for a mechanism-only or intentionally uncovered disposition, without relabelling the historical assertion as authoritative.
