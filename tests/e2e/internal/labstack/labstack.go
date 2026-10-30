//go:build e2e

// Package labstack is the import surface for AT harnesses. It delegates to the
// acceptance implementation so releases, enrollment and process hooks cannot
// drift into copied overlays. Import with -tags e2e.
package labstack

import "github.com/TED-H-com/Freedom-VPN/tests/e2e"

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

var Open = e2e.OpenStack
var PrepareServingBaseline = e2e.PrepareServingBaseline
var PrepareServingBaselineWithOptions = e2e.PrepareServingBaselineWithOptions
var RollbackAllowlist = e2e.RollbackAllowlist
