package fleetupdate

import "github.com/google/uuid"

type apiID = uuid.UUID

func equalID(a, b *uuid.UUID) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
