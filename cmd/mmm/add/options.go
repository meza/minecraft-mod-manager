package add

import (
	"errors"

	"github.com/spf13/cobra"
)

func readAddOptions(cmd *cobra.Command, args []string) (addOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return addOptions{}, err
	}
	nonInteractive, err := cmd.Flags().GetBool("non-interactive")
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

	return addOptions{
		Platform:             args[0],
		ProjectID:            args[1],
		ConfigPath:           configPath,
		NonInteractive:       nonInteractive,
		Quiet:                quiet,
		Debug:                debug,
		Version:              version,
		AllowVersionFallback: allowFallback,
	}, nil
}

func normalizeAddError(err error) error {
	if errors.Is(err, errAborted) {
		return nil
	}
	return err
}
