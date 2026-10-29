package audit

import (
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// SystemActor is the D-029 singleton for internal system work. Treat it as
// read-only; it is never populated from request JSON or an external identity.
var SystemActor = Actor{Type: api.AuditEventActorTypeSystem, ID: uuid.MustParse("00000000-0000-0000-0000-000000000003")}
