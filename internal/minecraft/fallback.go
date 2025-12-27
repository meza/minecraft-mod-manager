package minecraft

import (
	"context"
	"fmt"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
)

type InvalidReleaseVersionError struct {
	Version string
}

func (err *InvalidReleaseVersionError) Error() string {
	return fmt.Sprintf("minecraft version %q is not a known release", err.Version)
}

func (err *InvalidReleaseVersionError) Is(target error) bool {
	other, ok := target.(*InvalidReleaseVersionError)
	if !ok {
		return false
	}
	return other.Version == err.Version
}

// NextPatchDown returns the next lower patch release for the same series key.
// Mojang release sequences can omit ".0" releases (for example, 1.20 -> 1.20.1),
// so falling back from 1.20.1 to 1.20 is considered a patch-level fallback here.
// It errors when the provided version is not a release entry in the manifest.
func NextPatchDown(ctx context.Context, version string, client httpclient.Doer) (string, bool, error) {
	if strings.TrimSpace(version) == "" {
		return "", false, &InvalidReleaseVersionError{Version: version}
	}

	manifest, err := getMinecraftVersionManifest(ctx, client)
	if err != nil {
		return "", false, err
	}

	releaseVersions := releaseVersionsFromManifest(manifest)
	if len(releaseVersions) == 0 {
		return "", false, &InvalidReleaseVersionError{Version: version}
	}

	seriesKey, ok := seriesKeyFromVersion(version)
	if !ok {
		return "", false, &InvalidReleaseVersionError{Version: version}
	}

	startIndex := indexOfVersion(version, releaseVersions)
	if startIndex < 0 {
		return "", false, &InvalidReleaseVersionError{Version: version}
	}

	for index := startIndex + 1; index < len(releaseVersions); index++ {
		nextVersion := releaseVersions[index]
		nextSeriesKey, ok := seriesKeyFromVersion(nextVersion)
		if !ok {
			continue
		}
		if nextSeriesKey == seriesKey {
			return nextVersion, true, nil
		}
	}

	return version, false, nil
}

func releaseVersionsFromManifest(manifest *versionManifest) []string {
	if manifest == nil {
		return []string{}
	}

	versions := make([]string, 0, len(manifest.Versions))
	for _, manifestVersion := range manifest.Versions {
		if manifestVersion.Type != "release" {
			continue
		}
		versions = append(versions, manifestVersion.ID)
	}
	return versions
}

func seriesKeyFromVersion(version string) (string, bool) {
	segments := strings.Split(version, ".")
	if len(segments) < 2 {
		return "", false
	}
	return segments[0] + "." + segments[1], true
}

func indexOfVersion(version string, versions []string) int {
	for index, candidate := range versions {
		if candidate == version {
			return index
		}
	}
	return -1
}
