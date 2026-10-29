// Package netlab is the Gate G2 network lab. Scenario tests use the netlab
// build tag and skip unless FVPN_NETLAB=1 and the lab container is already
// running. Nothing in this package is safe to run on the host network
// namespace: topology changes happen only inside that container.
package netlab
