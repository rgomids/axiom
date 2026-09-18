package local

import (
	"crypto/rand"
	"fmt"

	"github.com/rgomids/axiom/internal/projectapp"
)

// IdentityAllocator uses operating-system entropy only when the application has
// already determined that a new Project must be created.
type IdentityAllocator struct{}

func (IdentityAllocator) NewID() (string, []projectapp.Issue) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", []projectapp.Issue{{Phase: projectapp.PersistencePhase, Field: projectapp.ProjectField, Code: projectapp.StoreFailure}}
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}
