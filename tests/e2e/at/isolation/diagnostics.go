package isolation

// UncreatedCloneClosure prevents collection commands against a clone whose
// creation stage was never reached. This is unavailable evidence, not a stop.
func UncreatedCloneClosure(container string, creationReached bool) map[string]any {
	if creationReached {
		return nil
	}
	return map[string]any{"container": container, "status": "not created", "stage": "clone_creation", "reason": "not created: setup exited before clone creation", "ok": false}
}
