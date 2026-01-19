package add

import (
	"errors"

	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/spf13/cobra"
)

func readAddOptions(cmd *cobra.Command, args []string) (addOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return addOptions{}, err
	}
	unattended, err := cmd.Flags().GetBool("unattended")
	if err != nil {
		return addOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return addOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return addOptions{}, err
	}
	version, err := cmd.Flags().GetString("version")
	if err != nil {
		return addOptions{}, err
	}
	allowFallback, err := cmd.Flags().GetBool("allow-version-fallback")
	if err != nil {
		return addOptions{}, err
	}
	lockSync, err := locksync.PolicyFlagsFromFlags(cmd.Flags())
	if err != nil {
		return addOptions{}, err
	}

	return addOptions{
		Platform:             args[0],
		ProjectID:            args[1],
		ConfigPath:           configPath,
		Unattended:           unattended,
		Quiet:                quiet,
		Debug:                debug,
		Version:              version,
		AllowVersionFallback: allowFallback,
		LockSync:             lockSync,
	}, nil
}

func normalizeAddError(err error) error {
	if errors.Is(err, errAborted) {
		return nil
	}
	return err
}
