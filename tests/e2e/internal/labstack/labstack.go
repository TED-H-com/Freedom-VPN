//go:build e2e

// Package labstack is the import surface for AT harnesses. It delegates to the
// acceptance implementation so releases, enrollment and process hooks cannot
// drift into copied overlays. Import with -tags e2e.
package labstack

import (
	"context"
	"net/http"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"
)

type NodeTelemetry = e2e.NodeTelemetry
type NodeTelemetrySample = e2e.NodeTelemetrySample
type NativeLogScan = e2e.NativeLogScan
type NativePeer = e2e.NativePeer
type NativeScale = e2e.NativeScale
type Stack = e2e.Stack
type BrowserSession = e2e.BrowserSession
type Response = e2e.Response
type BuiltRelease = e2e.BuiltRelease
type EnrolledNode = e2e.EnrolledNode
type ReleaseOptions = e2e.ReleaseOptions
type BaselineOptions = e2e.BaselineOptions
type ServingBaseline = e2e.ServingBaseline

func Open() (*Stack, error) { return e2e.OpenStack() }
func PrepareServingBaseline(ctx context.Context, client *http.Client, csrf string, candidates bool) (ServingBaseline, error) {
	return e2e.PrepareServingBaseline(ctx, client, csrf, candidates)
}
func PrepareServingBaselineWithOptions(ctx context.Context, client *http.Client, csrf string, options BaselineOptions) (ServingBaseline, error) {
	return e2e.PrepareServingBaselineWithOptions(ctx, client, csrf, options)
}
func RollbackAllowlist(lastGood uuid.UUID, allowed ...uuid.UUID) []uuid.UUID {
	return e2e.RollbackAllowlist(lastGood, allowed...)
}

type UpdaterProbe = e2e.UpdaterProbe
type ProbeObservation = e2e.ProbeObservation
type NativeStreamResult = e2e.NativeStreamResult
type EnrolledIdentityInput = e2e.EnrolledIdentityInput
type EnrolledIdentityResult = e2e.EnrolledIdentityResult
type EnrollmentPeers = e2e.EnrollmentPeers
type IdentityFingerprints = e2e.IdentityFingerprints
type ClonedIdentityFixture = e2e.ClonedIdentityFixture
type CloneObservation = e2e.CloneObservation
type ClonePresentation = e2e.ClonePresentation
type IngressCounter = e2e.IngressCounter
type MemberIngressResult = e2e.MemberIngressResult
type MemberIngressProof = e2e.MemberIngressProof
