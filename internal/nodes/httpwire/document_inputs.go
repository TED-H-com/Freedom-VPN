package httpwire

import (
	"context"
	"errors"
	"slices"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ResolveDocumentInputs supplies D-056/D-058 fields before the compiler hashes,
// compares or signs a document. Revision IDs are the compiler's DB policy snapshot.
func (r *Runtime) ResolveDocumentInputs(ctx context.Context, tx pgx.Tx, state *api.DesiredState, revisions []uuid.UUID) error {
	state.EgressProbe = presence.FromPointer(r.Config.EgressProbe)
	state.Dns.ResolverDocument.SetNull()
	profile, err := activeResolverProfile(ctx, tx)
	if err != nil || profile == uuid.Nil {
		return err
	}
	if r.ResolverDocuments == nil {
		return errors.New("resolver document source required")
	}
	document, err := r.ResolverDocuments.LatestPublished(ctx, profile)
	if errors.Is(err, pgx.ErrNoRows) {
		return apierr.New(apierr.DependencyUnavailable, "Node DNS profile has no published resolver document.")
	}
	if err != nil {
		return err
	}
	if err = nodes.ValidateResolverDocument(document); err != nil {
		return err
	}
	ref := document.Reference
	if !slices.Contains(revisions, ref.ProfileRevisionId) || state.PolicyRefs.DnsGeneration != ref.Generation {
		return errors.New("resolver publication changed during compilation")
	}
	state.Dns.ResolverDocument.Set(ref)
	return nil
}

func activeResolverProfile(ctx context.Context, q nodes.Querier) (uuid.UUID, error) {
	rows, err := q.Query(ctx, `SELECT id FROM policy_definitions WHERE type='dns_profile' AND active_revision_id IS NOT NULL ORDER BY id LIMIT 2`)
	if err != nil {
		return uuid.Nil, err
	}
	definitions, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return uuid.Nil, err
	}
	if len(definitions) == 0 {
		return uuid.Nil, nil
	}
	// There is no node-to-profile assignment in the current contract. Never
	// arbitrarily choose one profile if the fleet has ambiguous active profiles.
	if len(definitions) != 1 {
		return uuid.Nil, apierr.New(apierr.DependencyUnavailable, "Node DNS profile selection is ambiguous.")
	}
	return definitions[0], nil
}
