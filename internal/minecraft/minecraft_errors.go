package minecraft

import "errors"

var ErrManifestNotFound = errors.New("minecraft version manifest not found")
var ErrCouldNotDetermineLatestVersion = errors.New("could not determine latest version")
