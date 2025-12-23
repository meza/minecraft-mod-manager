package modrinth

import (
	"encoding/json"
	"net/http"
	"net/url"
)

var newRequestWithContext = http.NewRequestWithContext
var marshalJSON = json.Marshal
var parseURL = url.Parse
