package manifest

import "github.com/rgomids/axiom/internal/portableconfig"

// The codec delegates structural safety to the shared portable policy.
func safeValue(value, kind string) bool { return portableconfig.SafeValue(value, kind) }
