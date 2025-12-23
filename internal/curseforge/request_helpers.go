package curseforge

import (
	"encoding/json"
	"net/http"
)

var newRequestWithContext = http.NewRequestWithContext
var marshalJSON = json.Marshal
