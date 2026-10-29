package worker

// Jobs is the single composition point for domain jobs. Register a domain
// function here with registry.Register("domain.event", handler).
func Jobs() Registry {
	registry := Registry{}
	registerPoliciesJobs(registry)
	registerUsageJobs(registry)
	return registry
}
