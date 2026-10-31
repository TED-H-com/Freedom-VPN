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

type DeviceCounters = e2e.DeviceCounters
type EstablishedTransports = e2e.EstablishedTransports
type RelayPath = e2e.RelayPath
type UsageSnapshot = e2e.UsageSnapshot
type InstallerHashes = e2e.InstallerHashes
type InstallerOutcome = e2e.InstallerOutcome
type InstallerServiceState = e2e.InstallerServiceState
type ContainerExport = e2e.ContainerExport

type MemberTransportStatus = e2e.MemberTransportStatus
type DNSPathObservation = e2e.DNSPathObservation

type GrantTransition = e2e.GrantTransition
type GrantWireEvent = e2e.GrantWireEvent
type DNSQueryTrace = e2e.DNSQueryTrace

type EnrollingFleetSnapshot = e2e.EnrollingFleetSnapshot
type NativeFixtureInputs = e2e.NativeFixtureInputs
type NativeFixtureSelection = e2e.NativeFixtureSelection
type NativeUnderlayObservation = e2e.NativeUnderlayObservation
type NativeDNSConfiguration = e2e.NativeDNSConfiguration
type FixtureProbeResult = e2e.FixtureProbeResult

func SelectNativeFixture(in NativeFixtureInputs) (NativeFixtureSelection, error) {
	return e2e.SelectNativeFixture(in)
}

type FirstOfferEvidence = e2e.FirstOfferEvidence
type NamespaceRestartEvidence = e2e.NamespaceRestartEvidence

type SelectedNativeExit = e2e.SelectedNativeExit
