package scan

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	curseforgeFingerprint "github.com/meza/curseforge-fingerprint-go"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/mmmignore"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

type scanOptions struct {
	ConfigPath string
	Quiet      bool
	Debug      bool
	Prefer     string
	Add        bool
}

type scanDeps struct {
	fs              afero.Fs
	clients         platform.Clients
	minecraftClient httpclient.Doer
	logger          *logger.Logger
	prompter        prompter
	telemetry       func(telemetry.CommandTelemetry)

	curseforgeFingerprint      func(string) uint32
	modrinthVersionForSha      func(context.Context, string, httpclient.Doer) (*modrinth.Version, error)
	modrinthProjectTitle       func(context.Context, string, httpclient.Doer) (string, error)
	curseforgeFingerprintMatch func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error)
	curseforgeProjectName      func(context.Context, string, httpclient.Doer) (string, error)
}

type prompter interface {
	ConfirmAdd() (bool, error)
}

type terminalPrompter struct {
	in  io.Reader
	out io.Writer
}

func (prompter terminalPrompter) ConfirmAdd() (bool, error) {
	if _, err := fmt.Fprintf(prompter.out, "%s (y/N): ", i18n.T("cmd.scan.confirm_add")); err != nil {
		return false, err
	}
	answer, err := readLine(prompter.in)
	if err != nil {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func readLine(reader io.Reader) (string, error) {
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return scanner.Text(), nil
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: i18n.T("cmd.scan.short"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runScanCommand(cmd)
		},
	}

	cmd.Flags().StringP("prefer", "p", string(models.MODRINTH), i18n.T("cmd.scan.flag.prefer"))
	cmd.Flags().BoolP("add", "a", false, i18n.T("cmd.scan.flag.add"))

	return cmd
}

func runScanCommand(cmd *cobra.Command) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.scan")
	defer span.End()

	opts, err := scanOptionsFromFlags(cmd)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		return err
	}

	deps := defaultScanDeps(cmd, opts)
	payload, err := runScan(ctx, cmd, opts, deps)
	span.SetAttributes(attribute.Bool("success", err == nil))

	if deps.telemetry != nil {
		deps.telemetry(payload)
	}
	return err
}

func scanOptionsFromFlags(cmd *cobra.Command) (scanOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return scanOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return scanOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return scanOptions{}, err
	}
	prefer, err := cmd.Flags().GetString("prefer")
	if err != nil {
		return scanOptions{}, err
	}
	add, err := cmd.Flags().GetBool("add")
	if err != nil {
		return scanOptions{}, err
	}

	return scanOptions{
		ConfigPath: configPath,
		Quiet:      quiet,
		Debug:      debug,
		Prefer:     prefer,
		Add:        add,
	}, nil
}

func defaultScanDeps(cmd *cobra.Command, opts scanOptions) scanDeps {
	log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts.Quiet, opts.Debug)
	limiter := rate.NewLimiter(rate.Inf, 0)

	return scanDeps{
		fs:              afero.NewOsFs(),
		clients:         platform.DefaultClients(limiter),
		minecraftClient: httpclient.NewRLClient(limiter),
		logger:          log,
		prompter:        terminalPrompter{in: cmd.InOrStdin(), out: cmd.OutOrStdout()},
		telemetry:       telemetry.RecordCommand,

		curseforgeFingerprint:      curseforgeFingerprint.GetFingerprintFor,
		modrinthVersionForSha:      defaultModrinthVersionForSha,
		modrinthProjectTitle:       defaultModrinthProjectTitle,
		curseforgeFingerprintMatch: defaultCurseforgeFingerprintMatch,
		curseforgeProjectName:      defaultCurseforgeProjectName,
	}
}

type scanCandidate struct {
	Path     string
	FileName string
	Sha1     string
}

type scanMatch struct {
	Path        string
	Platform    models.Platform
	ProjectID   string
	Name        string
	FileName    string
	Hash        string
	ReleaseDate string
	DownloadURL string
}

type scanUnsure struct {
	Path  string
	Error error
}

//nolint:gocognit,gocyclo,funlen,nestif // Scan flow matches spec stages; splitting would obscure control flow.
func runScan(ctx context.Context, cmd *cobra.Command, opts scanOptions, deps scanDeps) (telemetry.CommandTelemetry, error) {
	meta := config.NewMetadata(opts.ConfigPath)
	setupCoordinator := modsetup.NewSetupCoordinator(deps.fs, deps.minecraftClient, nil)

	cfg, lock, err := setupCoordinator.EnsureConfigAndLock(ctx, meta, opts.Quiet)
	if err != nil {
		return scanFailureTelemetry(err), err
	}

	preferPlatform, err := resolvePreferredPlatform(opts.Prefer, deps.logger)
	if err != nil {
		return scanFailureTelemetry(err), err
	}

	files, err := listJarFiles(deps.fs, meta, cfg)
	if err != nil {
		return scanFailureTelemetry(err), err
	}

	unmanaged := unmanagedFiles(files, lock)

	if len(unmanaged) == 0 {
		deps.logger.Log(i18n.T("cmd.scan.all_managed"), false)
		return scanSuccessTelemetryWithoutArgs(), nil
	}

	candidates, err := sha1Candidates(ctx, deps.fs, unmanaged)
	if err != nil {
		return scanFailureTelemetry(err), err
	}

	matches, unknown, unsure := identifyCandidates(ctx, candidates, preferPlatform, deps)
	printResults(deps.logger, cmd.OutOrStdout(), preferPlatform, matches, unknown, unsure)

	shouldPersist, err := confirmPersist(opts, deps)
	if err != nil {
		return scanFailureTelemetry(err), err
	}

	if shouldPersist {
		if len(unsure) > 0 {
			deps.logger.Log(i18n.T("cmd.scan.persist_skipped_unsure"), false)
			return scanSuccessTelemetry(preferPlatform, opts.Add), nil
		}

		persisted, err := persistScanMatches(ctx, cmd, meta, setupCoordinator, deps, matches, cfg, lock)
		if err != nil {
			return scanFailureTelemetry(err), err
		}
		if persisted {
			deps.logger.Log(i18n.T("cmd.scan.persisted"), false)
		}
	}

	return scanSuccessTelemetry(preferPlatform, opts.Add), nil
}

func scanFailureTelemetry(err error) telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{Command: "scan", Success: false, ExitCode: 1, Error: err}
}

func scanSuccessTelemetry(preferPlatform models.Platform, add bool) telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{
		Command:  "scan",
		Success:  true,
		ExitCode: 0,
		Arguments: map[string]interface{}{
			"prefer": preferPlatform,
			"add":    add,
		},
	}
}

func scanSuccessTelemetryWithoutArgs() telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{
		Command:  "scan",
		Success:  true,
		ExitCode: 0,
	}
}

func resolvePreferredPlatform(value string, log *logger.Logger) (models.Platform, error) {
	preferPlatform := normalizePlatform(value)
	if preferPlatform != models.MODRINTH && preferPlatform != models.CURSEFORGE {
		platformErr := fmt.Errorf("unknown platform: %s", value)
		log.Error(platformErr.Error())
		return "", platformErr
	}
	return preferPlatform, nil
}

func unmanagedFiles(files []string, lock []models.ModInstall) []string {
	unmanaged := make([]string, 0, len(files))
	for _, file := range files {
		if fileIsManaged(file, lock) {
			continue
		}
		unmanaged = append(unmanaged, file)
	}
	return unmanaged
}

func confirmPersist(opts scanOptions, deps scanDeps) (bool, error) {
	shouldPersist := opts.Add
	if !shouldPersist && !opts.Quiet && deps.prompter != nil {
		confirm, err := deps.prompter.ConfirmAdd()
		if err != nil {
			return false, err
		}
		shouldPersist = confirm
	}
	return shouldPersist, nil
}

func persistScanMatches(ctx context.Context, cmd *cobra.Command, meta config.Metadata, setupCoordinator *modsetup.SetupCoordinator, deps scanDeps, matches []scanMatch, cfg models.ModsJSON, lock []models.ModInstall) (bool, error) {
	changedConfig := false
	changedLock := false

	for _, match := range matches {
		outcome, err := setupCoordinator.UpsertConfigAndLock(cfg, lock, match.Platform, match.ProjectID, platform.RemoteMod{
			Name:        match.Name,
			FileName:    match.FileName,
			Hash:        match.Hash,
			ReleaseDate: match.ReleaseDate,
			DownloadURL: match.DownloadURL,
		}, modsetup.EnsurePersistOptions{})
		if err != nil {
			deps.logger.Log(tui.ErrorIcon(tui.IsTerminalWriter(cmd.OutOrStdout()))+i18n.T("cmd.scan.persist_failed", i18n.Tvars{
				Data: &i18n.TData{"file": match.FileName},
			}), false)
			continue
		}

		if outcome.Result.ConfigAdded || outcome.Result.ConfigUpdated {
			changedConfig = true
		}
		if outcome.Result.LockAdded || outcome.Result.LockUpdated {
			changedLock = true
		}

		cfg = outcome.Config
		lock = outcome.Lock
	}

	if changedConfig {
		if err := config.WriteConfig(ctx, deps.fs, meta, cfg); err != nil {
			return false, err
		}
	}
	if changedLock {
		if err := config.WriteLock(ctx, deps.fs, meta, lock); err != nil {
			return false, err
		}
	}

	return changedConfig || changedLock, nil
}

func listJarFiles(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON) ([]string, error) {
	all, err := afero.ReadDir(fs, meta.ModsFolderPath(cfg))
	if err != nil {
		return nil, err
	}

	candidates := make([]string, 0, len(all))
	for _, entry := range all {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".jar") {
			continue
		}
		candidates = append(candidates, filepath.Join(meta.ModsFolderPath(cfg), entry.Name()))
	}

	patterns, err := mmmignore.ListPatterns(fs, meta.Dir())
	if err != nil {
		return nil, err
	}

	filtered := make([]string, 0, len(candidates))
	for _, path := range candidates {
		if mmmignore.IsIgnored(meta.Dir(), path, patterns) {
			continue
		}
		filtered = append(filtered, path)
	}
	return filtered, nil
}

func fileIsManaged(filePath string, installations []models.ModInstall) bool {
	filename := filepath.Base(filePath)
	for _, install := range installations {
		if install.FileName == filename {
			return true
		}
	}
	return false
}

func sha1Candidates(ctx context.Context, fs afero.Fs, files []string) ([]scanCandidate, error) {
	out := make([]scanCandidate, len(files))

	group, groupCtx := errgroup.WithContext(ctx)
	limit := runtime.GOMAXPROCS(0)
	group.SetLimit(limit)

	for i := range files {
		i := i
		group.Go(func() error {
			sha, err := sha1ForFile(groupCtx, fs, files[i])
			if err != nil {
				return err
			}
			out[i] = scanCandidate{
				Path:     files[i],
				FileName: filepath.Base(files[i]),
				Sha1:     sha,
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

var newSha1Hasher = sha1.New

func sha1ForFile(ctx context.Context, fs afero.Fs, path string) (hash string, returnErr error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	hasher := newSha1Hasher()
	buf := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, readErr := file.Read(buf)
		if n > 0 {
			if _, err := hasher.Write(buf[:n]); err != nil {
				return "", err
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return "", readErr
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func identifyCandidates(ctx context.Context, candidates []scanCandidate, prefer models.Platform, deps scanDeps) ([]scanMatch, []string, []scanUnsure) {
	preferredMatches, preferredMisses, preferUnsure := lookupOnPlatform(ctx, candidates, prefer, deps)

	fallback := alternatePlatform(prefer)
	fallbackMatches, fallbackMisses, fallbackUnsure := lookupOnPlatform(ctx, preferredMisses, fallback, deps)

	matches := make([]scanMatch, 0, len(preferredMatches)+len(fallbackMatches))
	matches = append(matches, preferredMatches...)
	matches = append(matches, fallbackMatches...)

	for path, err := range fallbackUnsure {
		preferUnsure[path] = err
	}
	for _, match := range matches {
		delete(preferUnsure, match.Path)
	}

	unknown := make([]string, 0, len(fallbackMisses))
	for _, miss := range fallbackMisses {
		if _, isUnsure := preferUnsure[miss.Path]; isUnsure {
			continue
		}
		unknown = append(unknown, miss.Path)
	}
	sort.Strings(unknown)

	unsure := make([]scanUnsure, 0, len(preferUnsure))
	for path, err := range preferUnsure {
		unsure = append(unsure, scanUnsure{Path: path, Error: err})
	}
	sort.SliceStable(unsure, func(i, j int) bool { return unsure[i].Path < unsure[j].Path })

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Platform != matches[j].Platform {
			return matches[i].Platform == prefer
		}
		if matches[i].Name != matches[j].Name {
			return matches[i].Name < matches[j].Name
		}
		return matches[i].FileName < matches[j].FileName
	})

	return matches, unknown, unsure
}

func lookupOnPlatform(ctx context.Context, candidates []scanCandidate, platformValue models.Platform, deps scanDeps) ([]scanMatch, []scanCandidate, map[string]error) {
	switch platformValue {
	case models.MODRINTH:
		return lookupModrinth(ctx, candidates, deps)
	case models.CURSEFORGE:
		return lookupCurseforge(ctx, candidates, deps)
	default:
		return nil, candidates, map[string]error{}
	}
}

func lookupModrinth(ctx context.Context, candidates []scanCandidate, deps scanDeps) ([]scanMatch, []scanCandidate, map[string]error) {
	results := make([]modrinthLookupResult, len(candidates))
	titleCache := newModrinthTitleCache()

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(4)

	for i := range candidates {
		i := i
		group.Go(func() error {
			if err := groupCtx.Err(); err != nil {
				return err
			}
			results[i] = lookupModrinthCandidate(groupCtx, candidates[i], deps, titleCache)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, candidates, matchErrorsForCandidates(candidates, err)
	}

	return splitModrinthResults(candidates, results)
}

type modrinthLookupResult struct {
	match *scanMatch
	err   error
	miss  bool
}

func lookupModrinthCandidate(ctx context.Context, candidate scanCandidate, deps scanDeps, titleCache *modrinthTitleCache) modrinthLookupResult {
	version, err := deps.modrinthVersionForSha(ctx, candidate.Sha1, deps.clients.Modrinth)
	if err != nil {
		var notFound *modrinth.VersionNotFoundError
		if errors.As(err, &notFound) {
			return modrinthLookupResult{miss: true}
		}
		return modrinthLookupResult{err: err}
	}

	projectID := version.ProjectID
	title, err := cachedModrinthTitle(ctx, projectID, deps, titleCache)
	if err != nil {
		return modrinthLookupResult{err: err}
	}

	downloadInfo, err := modrinthDownloadDetails(version)
	if err != nil {
		return modrinthLookupResult{err: err}
	}

	return modrinthLookupResult{match: &scanMatch{
		Path:        candidate.Path,
		Platform:    models.MODRINTH,
		ProjectID:   projectID,
		Name:        title,
		FileName:    candidate.FileName,
		Hash:        candidate.Sha1,
		ReleaseDate: downloadInfo.publishedAt,
		DownloadURL: downloadInfo.downloadURL,
	}}
}

type modrinthTitleFetch struct {
	ready chan struct{}
	title string
	err   error
}

type modrinthTitleCache struct {
	mu       sync.Mutex
	titles   map[string]string
	inflight map[string]*modrinthTitleFetch
}

func newModrinthTitleCache() *modrinthTitleCache {
	return &modrinthTitleCache{
		titles:   make(map[string]string),
		inflight: make(map[string]*modrinthTitleFetch),
	}
}

func (cache *modrinthTitleCache) get(ctx context.Context, projectID string, deps scanDeps) (string, error) {
	cache.mu.Lock()
	if title, ok := cache.titles[projectID]; ok {
		cache.mu.Unlock()
		return title, nil
	}
	if pending, ok := cache.inflight[projectID]; ok {
		cache.mu.Unlock()
		select {
		case <-pending.ready:
			cache.mu.Lock()
			title := pending.title
			err := pending.err
			cache.mu.Unlock()
			return title, err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	pending := &modrinthTitleFetch{ready: make(chan struct{})}
	cache.inflight[projectID] = pending
	cache.mu.Unlock()

	title, err := deps.modrinthProjectTitle(ctx, projectID, deps.clients.Modrinth)

	cache.mu.Lock()
	if err == nil {
		cache.titles[projectID] = title
	}
	pending.title = title
	pending.err = err
	delete(cache.inflight, projectID)
	close(pending.ready)
	cache.mu.Unlock()
	return title, err
}

func cachedModrinthTitle(ctx context.Context, projectID string, deps scanDeps, titleCache *modrinthTitleCache) (string, error) {
	return titleCache.get(ctx, projectID, deps)
}

func matchErrorsForCandidates(candidates []scanCandidate, err error) map[string]error {
	unsure := make(map[string]error, len(candidates))
	for _, candidate := range candidates {
		unsure[candidate.Path] = err
	}
	return unsure
}

func splitModrinthResults(candidates []scanCandidate, results []modrinthLookupResult) ([]scanMatch, []scanCandidate, map[string]error) {
	matches := make([]scanMatch, 0, len(candidates))
	misses := make([]scanCandidate, 0, len(candidates))
	unsure := make(map[string]error)

	for i, res := range results {
		candidate := candidates[i]
		if res.match != nil {
			matches = append(matches, *res.match)
			continue
		}
		if res.miss {
			misses = append(misses, candidate)
			continue
		}
		if res.err != nil {
			unsure[candidate.Path] = res.err
		}
	}

	return matches, misses, unsure
}

func lookupCurseforge(ctx context.Context, candidates []scanCandidate, deps scanDeps) ([]scanMatch, []scanCandidate, map[string]error) {
	matches := make([]scanMatch, 0)
	unsure := make(map[string]error)

	fingerprintIndex := buildCurseforgeFingerprintIndex(candidates, deps)

	sort.Ints(fingerprintIndex.fingerprints)
	unique := uniqueInts(fingerprintIndex.fingerprints)
	if len(unique) == 0 {
		return nil, candidates, unsure
	}

	result, err := deps.curseforgeFingerprintMatch(ctx, unique, deps.clients.Curseforge)
	if err != nil {
		return nil, nil, buildCurseforgeErrors(candidates, fingerprintIndex.fingerprintByIndex, err)
	}

	nameCache := make(map[string]string)
	var nameMu sync.Mutex

	addCurseforgeMatches(ctx, curseforgeMatchContext{
		candidates:           candidates,
		fingerprintToIndices: fingerprintIndex.fingerprintToIndices,
		matches:              result.Matches,
		deps:                 deps,
		nameCache:            nameCache,
		nameMu:               &nameMu,
		scanMatches:          &matches,
		unsure:               unsure,
	})
	misses := curseforgeMisses(candidates, matches, unsure)
	return matches, misses, unsure
}

type curseforgeFingerprintIndex struct {
	fingerprints         []int
	fingerprintByIndex   []int
	fingerprintToIndices map[int][]int
}

func buildCurseforgeFingerprintIndex(candidates []scanCandidate, deps scanDeps) curseforgeFingerprintIndex {
	fingerprints := make([]int, 0, len(candidates))
	fingerprintByIndex := make([]int, len(candidates))
	fingerprintToIndices := make(map[int][]int, len(candidates))

	for i, candidate := range candidates {
		fingerprint := int(deps.curseforgeFingerprint(candidate.Path))
		fingerprints = append(fingerprints, fingerprint)
		fingerprintByIndex[i] = fingerprint
		fingerprintToIndices[fingerprint] = append(fingerprintToIndices[fingerprint], i)
	}

	return curseforgeFingerprintIndex{
		fingerprints:         fingerprints,
		fingerprintByIndex:   fingerprintByIndex,
		fingerprintToIndices: fingerprintToIndices,
	}
}

func buildCurseforgeErrors(candidates []scanCandidate, fingerprintByIndex []int, err error) map[string]error {
	reason := curseforgeFingerprintFailureReason(err)
	unsure := make(map[string]error, len(candidates))
	for i, candidate := range candidates {
		unsure[candidate.Path] = fmt.Errorf("curseforge fingerprint %d: %s", fingerprintByIndex[i], reason)
	}
	return unsure
}

type curseforgeMatchContext struct {
	candidates           []scanCandidate
	fingerprintToIndices map[int][]int
	matches              []curseforge.File
	deps                 scanDeps
	nameCache            map[string]string
	nameMu               *sync.Mutex
	scanMatches          *[]scanMatch
	unsure               map[string]error
}

func addCurseforgeMatches(ctx context.Context, matchContext curseforgeMatchContext) {
	for _, file := range matchContext.matches {
		indices := matchContext.fingerprintToIndices[file.Fingerprint]
		if len(indices) == 0 {
			continue
		}

		projectID := fmt.Sprintf("%d", file.ProjectID)
		name, err := cachedCurseforgeProjectName(ctx, projectID, matchContext.deps, matchContext.nameCache, matchContext.nameMu)
		if err != nil {
			for _, index := range indices {
				matchContext.unsure[matchContext.candidates[index].Path] = err
			}
			continue
		}

		if strings.TrimSpace(file.DownloadURL) == "" {
			for _, index := range indices {
				matchContext.unsure[matchContext.candidates[index].Path] = errors.New("curseforge match missing download url")
			}
			continue
		}

		published := file.FileDate.Format(time.RFC3339)
		for _, index := range indices {
			*matchContext.scanMatches = append(*matchContext.scanMatches, scanMatch{
				Path:        matchContext.candidates[index].Path,
				Platform:    models.CURSEFORGE,
				ProjectID:   projectID,
				Name:        name,
				FileName:    matchContext.candidates[index].FileName,
				Hash:        matchContext.candidates[index].Sha1,
				ReleaseDate: published,
				DownloadURL: file.DownloadURL,
			})
		}
	}
}

func cachedCurseforgeProjectName(ctx context.Context, projectID string, deps scanDeps, nameCache map[string]string, nameMu *sync.Mutex) (string, error) {
	nameMu.Lock()
	name, ok := nameCache[projectID]
	nameMu.Unlock()
	if ok {
		return name, nil
	}

	name, err := deps.curseforgeProjectName(ctx, projectID, deps.clients.Curseforge)
	if err != nil {
		return "", err
	}

	nameMu.Lock()
	nameCache[projectID] = name
	nameMu.Unlock()
	return name, nil
}

func curseforgeMisses(candidates []scanCandidate, matches []scanMatch, unsure map[string]error) []scanCandidate {
	matched := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		matched[match.Path] = struct{}{}
	}

	misses := make([]scanCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := matched[candidate.Path]; ok {
			continue
		}
		if _, ok := unsure[candidate.Path]; ok {
			continue
		}
		misses = append(misses, candidate)
	}
	return misses
}

func curseforgeFingerprintFailureReason(err error) string {
	var apiError *curseforge.FingerprintAPIError
	if errors.As(err, &apiError) {
		err = apiError.Unwrap()
	}

	reason := err.Error()
	if strings.Contains(reason, "unexpected status code: 403") {
		return reason + " (check CURSEFORGE_API_KEY)"
	}
	return reason
}

type modrinthDownloadInfo struct {
	downloadURL string
	publishedAt string
}

func modrinthDownloadDetails(version *modrinth.Version) (modrinthDownloadInfo, error) {
	if version == nil {
		return modrinthDownloadInfo{}, errors.New("modrinth version is nil")
	}
	if len(version.Files) == 0 {
		return modrinthDownloadInfo{}, errors.New("modrinth version has no files")
	}

	chosen := version.Files[0]
	for _, file := range version.Files {
		if file.Primary {
			chosen = file
			break
		}
	}

	if strings.TrimSpace(chosen.URL) == "" {
		return modrinthDownloadInfo{}, errors.New("modrinth file missing url")
	}

	return modrinthDownloadInfo{downloadURL: chosen.URL, publishedAt: version.DatePublished.Format(time.RFC3339)}, nil
}

func uniqueInts(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizePlatform(value string) models.Platform {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(models.CURSEFORGE):
		return models.CURSEFORGE
	case string(models.MODRINTH):
		return models.MODRINTH
	default:
		return models.Platform(strings.ToLower(value))
	}
}

func alternatePlatform(platform models.Platform) models.Platform {
	if platform == models.CURSEFORGE {
		return models.MODRINTH
	}
	return models.CURSEFORGE
}

func printResults(log *logger.Logger, out io.Writer, _ models.Platform, matches []scanMatch, unknown []string, unsure []scanUnsure) {
	colorize := tui.IsTerminalWriter(out)

	if len(matches) > 0 {
		log.Log(i18n.T("cmd.scan.recognized.header"), false)
		for _, match := range matches {
			name := match.Name
			if colorize {
				name = tui.TitleStyle.Copy().Bold(true).Render(name)
			}
			log.Log(messageWithIcon(tui.SuccessIcon(colorize), i18n.T("cmd.scan.recognized.entry", i18n.Tvars{
				Data: &i18n.TData{
					"name":     name,
					"platform": match.Platform,
					"id":       match.ProjectID,
					"file":     match.FileName,
				},
			})), false)
		}
	}

	if len(unknown) > 0 {
		log.Log(i18n.T("cmd.scan.unknown.header"), false)
		for _, file := range unknown {
			log.Log(messageWithIcon(tui.ErrorIcon(colorize), i18n.T("cmd.scan.unknown.entry", i18n.Tvars{
				Data: &i18n.TData{"file": filepath.Base(file)},
			})), false)
		}
	}

	if len(unsure) > 0 {
		log.Log(i18n.T("cmd.scan.unsure.header"), false)
		for _, item := range unsure {
			reason := "unknown error"
			if item.Error != nil {
				reason = item.Error.Error()
			}
			log.Log(messageWithIcon(tui.ErrorIcon(colorize), i18n.T("cmd.scan.unsure.entry_with_reason", i18n.Tvars{
				Data: &i18n.TData{
					"file":   filepath.Base(item.Path),
					"reason": reason,
				},
			})), false)
		}
	}

	if len(matches) == 0 && len(unknown) == 0 && len(unsure) == 0 {
		log.Log(i18n.T("cmd.scan.no_results"), false)
	}
}

func defaultModrinthVersionForSha(ctx context.Context, sha1 string, doer httpclient.Doer) (*modrinth.Version, error) {
	client := modrinth.NewClient(doer)
	return modrinth.GetVersionForHash(ctx, modrinth.NewVersionHashLookup(sha1, modrinth.SHA1), client)
}

func defaultModrinthProjectTitle(ctx context.Context, projectID string, doer httpclient.Doer) (string, error) {
	client := modrinth.NewClient(doer)
	project, err := modrinth.GetProject(ctx, projectID, client)
	if err != nil {
		return "", err
	}
	return project.Title, nil
}

func defaultCurseforgeFingerprintMatch(ctx context.Context, fingerprints []int, doer httpclient.Doer) (*curseforge.FingerprintResult, error) {
	client := curseforge.NewClient(doer)
	return curseforge.GetFingerprintsMatches(ctx, fingerprints, client)
}

func defaultCurseforgeProjectName(ctx context.Context, projectID string, doer httpclient.Doer) (string, error) {
	client := curseforge.NewClient(doer)
	project, err := curseforge.GetProject(ctx, projectID, client)
	if err != nil {
		return "", err
	}
	return project.Name, nil
}
