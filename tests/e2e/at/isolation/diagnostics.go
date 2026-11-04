package isolation

// UncreatedCloneClosure prevents collection commands against a clone whose
// creation stage was never reached. This is unavailable evidence, not a stop.
func UncreatedCloneClosure(container string, creationReached bool) map[string]any {
	if creationReached {
		return nil
	}
	return map[string]any{"container": container, "status": "not created", "stage": "clone_creation", "reason": "not created: setup exited before clone creation", "ok": false}
}

// PublicationNote retains the command receipt even when successful setup does
// not add its temporary check to the acceptance rows. Every calling check also
// keeps its own full observations, including restoration commands.
func PublicationNote(check *Check, record *Record) func(string, string) {
	return func(name, detail string) {
		observation := Observation{Name: name, Detail: detail}
		check.Observations = append(check.Observations, observation)
		if name == "node_write" {
			record.Observations = append(record.Observations, observation)
		}
	}
}
