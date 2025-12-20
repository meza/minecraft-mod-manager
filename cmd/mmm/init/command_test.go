package init

import (
	"context"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestCommandWithRunner_ParsesFlags(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	var gotOptions initOptions
	cmd := commandWithRunner(func(_ context.Context, _ *cobra.Command, options initOptions, _ initDeps, _ config.Metadata) error {
		gotOptions = options
		return nil
	})
	addPersistentFlagsForTesting(cmd)
	setCommandOutputForTesting(cmd)

	cmd.SetArgs([]string{
		"--loader", "fabric",
		"--game-version", "1.21.1",
		"--release-types=release,beta",
		"--mods-folder", "mods",
	})

	assert.NoError(t, cmd.Execute())
	assert.Equal(t, models.FABRIC, gotOptions.Loader)
	assert.Equal(t, "1.21.1", gotOptions.GameVersion)
	assert.Equal(t, []models.ReleaseType{models.Release, models.Beta}, gotOptions.ReleaseTypes)
	assert.Equal(t, "mods", gotOptions.ModsFolder)
	assert.True(t, gotOptions.Provided.Loader)
	assert.True(t, gotOptions.Provided.GameVersion)
	assert.True(t, gotOptions.Provided.ReleaseTypes)
	assert.True(t, gotOptions.Provided.ModsFolder)
}

func TestCommandReturnsCommand(t *testing.T) {
	assert.NotNil(t, Command())
}

func TestCommandWithRunnerMissingGameVersionFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error {
		return nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, nil))
}

func TestCommandWithRunnerMissingModsFolderFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error {
		return nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("game-version", "latest", "game version")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, nil))
}

func TestCommandWithRunnerMissingReleaseTypesFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error {
		return nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("game-version", "latest", "game version")
	cmd.Flags().String("mods-folder", "mods", "mods folder")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, nil))
}

func TestCommandWithRunnerMissingConfigFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error {
		return nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("game-version", "latest", "game version")
	cmd.Flags().String("mods-folder", "mods", "mods folder")
	cmd.Flags().StringSlice("release-types", []string{"release"}, "release types")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, nil))
}

func TestCommandWithRunnerMissingQuietFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error {
		return nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("game-version", "latest", "game version")
	cmd.Flags().String("mods-folder", "mods", "mods folder")
	cmd.Flags().StringSlice("release-types", []string{"release"}, "release types")
	cmd.Flags().String("config", "modlist.json", "config")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, nil))
}

func TestCommandWithRunnerMissingDebugFlagErrors(t *testing.T) {
	runE := commandWithRunner(func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error {
		return nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("game-version", "latest", "game version")
	cmd.Flags().String("mods-folder", "mods", "mods folder")
	cmd.Flags().StringSlice("release-types", []string{"release"}, "release types")
	cmd.Flags().String("config", "modlist.json", "config")
	cmd.Flags().Bool("quiet", false, "quiet")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, nil))
}

func TestCommandWithRunnerInvalidReleaseTypesErrors(t *testing.T) {
	runE := commandWithRunner(func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error {
		return nil
	}).RunE

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("game-version", "latest", "game version")
	cmd.Flags().String("mods-folder", "mods", "mods folder")
	cmd.Flags().StringSlice("release-types", []string{"nope"}, "release types")
	cmd.Flags().String("config", "modlist.json", "config")
	cmd.Flags().Bool("quiet", false, "quiet")
	cmd.Flags().Bool("debug", false, "debug")
	setCommandOutputForTesting(cmd)

	assert.Error(t, runE(cmd, nil))
}

func TestGetAllReleaseTypesAndLoaders(t *testing.T) {
	releaseTypes := getAllReleaseTypes()
	assert.Contains(t, releaseTypes, "release")

	loaders := getAllLoaders()
	assert.Contains(t, loaders, "fabric")
}

func TestLoaderFlagStringAndType(t *testing.T) {
	flag := loaderFlag{value: models.FABRIC}
	assert.Equal(t, "fabric", flag.String())
	assert.Equal(t, "loader", flag.Type())
}

func TestCompleteLoadersAndReleaseTypes(t *testing.T) {
	loaderSuggestions, directive := completeLoaders(nil, nil, "")
	assert.NotEmpty(t, loaderSuggestions)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)

	releaseSuggestions, directive := completeReleaseTypes(nil, nil, "")
	assert.NotEmpty(t, releaseSuggestions)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestParseReleaseTypesRejectsEmpty(t *testing.T) {
	_, err := parseReleaseTypes([]string{" "})
	assert.ErrorContains(t, err, "release types cannot be empty")
}

func addPersistentFlagsForTesting(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("config", "c", "./modlist.json", "An alternative JSON file containing the configuration")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress all output")
	cmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug messages")
}

func setCommandOutputForTesting(cmd *cobra.Command) {
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
}
