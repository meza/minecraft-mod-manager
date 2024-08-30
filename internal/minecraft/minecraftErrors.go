package minecraft

import "errors"

var ManifestNotFound = errors.New("minecraft version manifest not found")
var CouldNotDetermineLatestVersion = errors.New("could not determine latest version")
