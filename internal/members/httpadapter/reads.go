package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func value[T any](p *T) T {
	if p != nil {
		return *p
	}
	var zero T
	return zero
}
func pointer[T any](v T) *T { return &v }
func nullableText(s string) presence.Nullable[string] {
	if s == "" {
		return presence.NewNullNullable[string]()
	}
	return presence.NewNullableWithValue(s)
}
func email(s *string) presence.Nullable[openapi_types.Email] {
	if s == nil {
		return presence.NewNullNullable[openapi_types.Email]()
	}
	return presence.NewNullableWithValue(openapi_types.Email(*s))
}

func (h Handlers) entitlement(ctx context.Context, m members.Member) (api.EntitlementSummary, error) {
	e := api.EntitlementSummary{State: api.EntitlementState(m.Status), ComputedAt: m.CreatedAt.UTC(), AdminSuspended: m.AdminSuspended,
		DeviceLimit: presence.NewNullNullable[int](), Plan: presence.NewNullNullable[api.PlanRef](),
		EffectiveServiceEnd: presence.NewNullNullable[time.Time](), GraceUntil: presence.NewNullNullable[time.Time](), PaidThrough: presence.NewNullNullable[time.Time](), ValidFrom: presence.NewNullNullable[time.Time]()}
	p := m.Entitlement
	if p == nil {
		return e, nil
	}
	e.State = api.EntitlementState(p.State)
	e.Revision = p.Revision
	e.ComputedAt = p.ComputedAt.UTC()
	e.AdminSuspended = p.AdminSuspended
	e.CancelAtPeriodEnd = p.CancelAtPeriodEnd
	e.EffectiveServiceEnd = nullableTime(p.EffectiveServiceEnd)
	e.GraceUntil = nullableTime(p.GraceUntil)
	e.PaidThrough = nullableTime(p.PaidThrough)
	e.ValidFrom = nullableTime(p.ValidFrom)
	// A stale projection never advertises usable service after its deadline (§9.2).
	s, err := h.services(ctx)
	if err != nil {
		return e, err
	}
	if (p.State == "active" || p.State == "grace" || p.State == "cancel_at_period_end") && !p.Allows(s.Members.Entitlements.Time()) {
		e.State = api.EntitlementStateExpired
	}
	if p.Plan != nil {
		e.DeviceLimit.Set(p.DeviceLimit)
		var ref api.PlanRef
		var raw []byte
		err := h.Pool.QueryRow(ctx, `SELECT p.id,p.code,r.payload FROM plans p JOIN plan_revisions r ON r.plan_id=p.id WHERE r.id=$1`, p.Plan.ID).Scan(&ref.PlanId, &ref.Code, &raw)
		if err != nil {
			return e, err
		}
		var payload struct {
			DisplayName api.LocalizedText `json:"display_name"`
		}
		if err = json.Unmarshal(raw, &payload); err != nil {
			return e, err
		}
		ref.PlanRevisionId = p.Plan.ID
		ref.DisplayName = payload.DisplayName
		if ref.DisplayName.ZhTW == "" {
			ref.DisplayName.ZhTW = ref.Code
		}
		e.Plan.Set(ref)
	}
	return e, nil
}
func subscription(m members.Member) presence.Nullable[api.SubscriptionSummary] {
	if m.Subscription == nil {
		return presence.NewNullNullable[api.SubscriptionSummary]()
	}
	s := m.Subscription
	status := api.SubscriptionSummaryReconciliationStatusOk
	if s.LastReconciledAt == nil {
		status = api.SubscriptionSummaryReconciliationStatusPending
	}
	if len(s.Anomalies) > 0 {
		status = api.SubscriptionSummaryReconciliationStatusAnomaly
	}
	return presence.NewNullableWithValue(api.SubscriptionSummary{Provider: s.Provider, ProviderStatus: s.ProviderStatus, CancelAtPeriodEnd: s.CancelAtPeriodEnd, PaidThrough: nullableTime(s.PaidThrough), LastReconciledAt: nullableTime(s.LastReconciledAt), ReconciliationStatus: status})
}
func (h Handlers) summary(ctx context.Context, m members.Member) (api.MemberSummary, error) {
	e, err := h.entitlement(ctx, m)
	if err != nil {
		return api.MemberSummary{}, err
	}
	u, err := usage.NewService(usage.Config{Pool: h.Pool})
	if err != nil {
		return api.MemberSummary{}, err
	}
	total, err := u.GetMemberUsageSummary(ctx, m.ID, "", nil)
	if err != nil {
		return api.MemberSummary{}, err
	}
	return api.MemberSummary{MemberId: m.ID, DisplayName: nullableText(m.DisplayName), Email: email(m.Email), AccountStatus: api.MemberSummaryAccountStatus(accountStatus(m.Status)), CreatedAt: m.CreatedAt.UTC(), ResourceVersion: m.ResourceVersion, AuthorizedDeviceCount: m.AuthorizedDevices, Entitlement: e, Usage: total}, nil
}
func (h Handlers) detail(ctx context.Context, m members.Member) (api.MemberDetail, error) {
	summary, err := h.summary(ctx, m)
	if err != nil {
		return api.MemberDetail{}, err
	}
	out := api.MemberDetail{MemberId: m.ID, DisplayName: summary.DisplayName, Email: summary.Email, AccountStatus: api.MemberDetailAccountStatus(accountStatus(m.Status)), CreatedAt: summary.CreatedAt, ClosedAt: nullableTime(m.ClosedAt), ResourceVersion: m.ResourceVersion, AuthorizedDeviceCount: m.AuthorizedDevices, Entitlement: summary.Entitlement, Usage: summary.Usage, Subscription: subscription(m), Notes: nullableText(m.Notes), MemberGroupIds: append([]uuid.UUID{}, m.Groups...), Grants: []api.EntitlementGrant{}}
	out.Suspension.Suspended = m.AdminSuspended
	out.Suspension.Reason = presence.FromPointer(m.SuspensionReason)
	out.Suspension.SuspendedAt = nullableTime(m.SuspendedAt)
	out.Suspension.SuspendedBy = presence.FromPointer(m.SuspendedBy)
	for _, g := range m.Grants {
		creator, err := uuid.Parse(g.CreatedBy)
		if err != nil {
			return out, err
		}
		out.Grants = append(out.Grants, api.EntitlementGrant{GrantId: g.ID, MemberId: m.ID, PlanRevisionId: presence.FromPointer(g.PlanRevisionID), Source: api.GrantSource(g.Source), Reason: g.Reason, ValidFrom: g.ValidFrom.UTC(), ValidUntil: g.ValidUntil.UTC(), RevokedAt: nullableTime(g.RevokedAt), CreatedAt: g.CreatedAt.UTC(), CreatedBy: creator})
	}
	return out, nil
}
func (h Handlers) ListMembers(ctx context.Context, r gen.ListMembersRequestObject) (gen.ListMembersResponseObject, error) {
	s, err := h.services(ctx)
	if err != nil {
		return nil, err
	}
	page, err := s.Members.ListFiltered(ctx, actor(ctx), value(r.Params.Cursor), value(r.Params.Limit), string(value(r.Params.EntitlementState)))
	if err != nil {
		if errors.Is(err, entitlement.ErrInvalid) {
			return nil, apierr.New(apierr.MalformedRequest, "Invalid member page.")
		}
		return nil, domainError(err)
	}
	body := api.MemberList{Items: []api.MemberSummary{}, NextCursor: nullableText(page.NextCursor)}
	for _, m := range page.Items {
		v, e := h.summary(ctx, m)
		if e != nil {
			return nil, e
		}
		body.Items = append(body.Items, v)
	}
	return gen.ListMembers200JSONResponse{Body: body}, nil
}
func (h Handlers) GetMember(ctx context.Context, r gen.GetMemberRequestObject) (gen.GetMemberResponseObject, error) {
	s, err := h.services(ctx)
	if err != nil {
		return nil, err
	}
	m, err := s.Members.Get(ctx, actor(ctx), r.MemberId)
	if err != nil {
		return nil, domainError(err)
	}
	body, err := h.detail(ctx, m)
	if err != nil {
		return nil, err
	}
	return gen.GetMember200JSONResponse{Body: body, Headers: gen.GetMember200ResponseHeaders{ETag: pointer(httpapi.ETag(m.ResourceVersion))}}, nil
}
func (h Handlers) GetMemberSelf(ctx context.Context, _ gen.GetMemberSelfRequestObject) (gen.GetMemberSelfResponseObject, error) {
	s, err := h.services(ctx)
	if err != nil {
		return nil, err
	}
	a := actor(ctx)
	m, err := s.Members.Get(ctx, a, a.ID)
	if err != nil {
		return nil, domainError(err)
	}
	e, err := h.entitlement(ctx, m)
	if err != nil {
		return nil, err
	}
	out := api.MemberSelf{MemberId: m.ID, DisplayName: nullableText(m.DisplayName), Email: email(m.Email), Entitlement: e, Subscription: subscription(m), FairUseResetAt: presence.NewNullNullable[time.Time](), NextActions: []api.NextAction{}}
	out.Limits.AuthorizedDeviceCount = m.AuthorizedDevices
	out.Limits.DeviceLimit = e.DeviceLimit
	groups, err := entitlement.ServingExitGroups(ctx, h.Pool, m.ID, s.Members.Entitlements.Time())
	if err != nil {
		return nil, err
	}
	if m.Entitlement != nil && len(groups) > 0 {
		err = h.Pool.QueryRow(ctx, `SELECT count(DISTINCT e.id) FROM exits e JOIN exit_groups eg ON eg.exit_id=e.id JOIN nodes n ON n.id=e.node_id WHERE eg.group_id=ANY($1) AND e.publication_intent='published' AND n.lifecycle_state='registered'`, groups).Scan(&out.Limits.ExitCount)
		if err != nil {
			return nil, err
		}
		now := s.Members.Entitlements.Time()
		out.FairUseResetAt.Set(time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC))
		if m.AuthorizedDevices >= m.Entitlement.DeviceLimit {
			out.NextActions = append(out.NextActions, api.NextActionFreeDeviceSlot)
		} else {
			out.NextActions = append(out.NextActions, api.NextActionInstallApp, api.NextActionRegisterDevice)
		}
	} else {
		switch e.State {
		case "pending_payment":
			out.NextActions = append(out.NextActions, api.NextActionCompletePayment)
		case "suspended", "closed":
			out.NextActions = append(out.NextActions, api.NextActionContactSupport)
		default:
			out.NextActions = append(out.NextActions, api.NextActionRenewSubscription)
		}
	}
	var pending bool
	if err = h.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM billing_sessions WHERE member_id=$1 AND kind='checkout' AND created_at>clock_timestamp()-interval '23 hours')`, m.ID).Scan(&pending); err != nil {
		return nil, err
	}
	if e.State == "pending_payment" && pending {
		out.NextActions = []api.NextAction{api.NextActionWaitForPaymentConfirmation}
	}
	return gen.GetMemberSelf200JSONResponse{Body: out}, nil
}
func (h Handlers) ListMemberDevices(ctx context.Context, r gen.ListMemberDevicesRequestObject) (gen.ListMemberDevicesResponseObject, error) {
	s, err := h.services(ctx)
	if err != nil {
		return nil, err
	}
	a := actor(ctx)
	page, err := s.Devices.List(ctx, a, a.ID, value(r.Params.Cursor), value(r.Params.Limit))
	if err != nil {
		if errors.Is(err, entitlement.ErrInvalid) {
			return nil, apierr.New(apierr.MalformedRequest, "Invalid device page.")
		}
		return nil, domainError(err)
	}
	out := api.MemberDeviceList{Items: []api.MemberDevice{}, NextCursor: nullableText(page.NextCursor), AuthorizedCount: page.AuthorizedCount, DeviceLimit: presence.NewNullNullable[int]()}
	if page.DeviceLimit > 0 {
		out.DeviceLimit.Set(page.DeviceLimit)
	}
	for _, d := range page.Items {
		out.Items = append(out.Items, memberDevice(d))
	}
	return gen.ListMemberDevices200JSONResponse{Body: out}, nil
}

func (h Handlers) ListAdminDevices(ctx context.Context, r gen.ListAdminDevicesRequestObject) (gen.ListAdminDevicesResponseObject, error) {
	s, err := h.services(ctx)
	if err != nil {
		return nil, err
	}
	page, err := s.Devices.ListAdmin(ctx, actor(ctx), devices.AdminQuery{Member: r.Params.MemberId, Unconfirmed: r.Params.AuthorizationUnconfirmed, Status: string(value(r.Params.Status)), Cursor: value(r.Params.Cursor), Limit: value(r.Params.Limit)})
	if err != nil {
		if errors.Is(err, entitlement.ErrInvalid) {
			return nil, apierr.New(apierr.MalformedRequest, "Invalid device page.")
		}
		return nil, domainError(err)
	}
	body := api.AdminDeviceList{Items: []api.AdminDevice{}, NextCursor: nullableText(page.NextCursor)}
	for _, d := range page.Items {
		item := memberDevice(d)
		body.Items = append(body.Items, api.AdminDevice{
			AuthorizationUnconfirmed:      item.AuthorizationUnconfirmed,
			AuthorizationUnconfirmedSince: nullableTime(d.FencedAt),
			AuthorizedAt:                  item.AuthorizedAt,
			CreatedAt:                     item.CreatedAt,
			DeviceId:                      item.DeviceId,
			LastSeenAt:                    item.LastSeenAt,
			MemberId:                      pointer(d.MemberID),
			Name:                          item.Name,
			Platform:                      api.AdminDevicePlatform(d.Platform),
			RevokedAt:                     item.RevokedAt,
			Status:                        item.Status,
		})
	}
	return gen.ListAdminDevices200JSONResponse{Body: body}, nil
}

func memberDevice(d devices.Device) api.MemberDevice {
	return api.MemberDevice{
		DeviceId:                 d.ID,
		Name:                     d.Name,
		Platform:                 api.MemberDevicePlatform(d.Platform),
		Status:                   api.DeviceStatus(d.PublicStatus()),
		AuthorizationUnconfirmed: pointer(d.AuthorizationUnconfirmed),
		CreatedAt:                d.CreatedAt.UTC(),
		AuthorizedAt:             nullableTime(d.AuthorizedAt),
		RevokedAt:                nullableTime(d.RevokedAt),
		LastSeenAt:               nullableTime(d.LastSeenAt),
	}
}

func nullableTime(t *time.Time) presence.Nullable[time.Time] {
	if t == nil {
		return presence.NewNullNullable[time.Time]()
	}
	return presence.NewNullableWithValue(t.UTC())
}

// Account closure and service rights are separate dimensions in the contract.
func accountStatus(status string) string {
	if status == "closed" {
		return "closed"
	}
	return "active"
}
