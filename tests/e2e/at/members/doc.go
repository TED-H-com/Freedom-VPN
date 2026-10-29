// Package members holds the Wave 4 member acceptance startup gate.
//
// The assigned stack cannot start on this revision: its port-base validator
// rejects 19000. The tagged gate reproduces that prerequisite failure through
// the existing prepare command and writes blocked evidence for all assigned
// cases. It does not replace the dev-stack runner or claim live AT coverage.
package members
