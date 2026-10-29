// Package updater is the node updater (§13.3 to §13.9, §13.11). It downloads
// a signed release, verifies it, and switches to it through the update
// journal. A crash resumes from the last durable stage. Member forwarding
// stays off while a transaction is applying or checking, and stays off after
// recovery_required.
//
// The updater never executes files from a manifest. It never changes
// publication intent, and it never deletes identity or the usage outbox.
// Local success is recorded in the result outbox until central acceptance;
// it is not itself acceptance (§13.6 step 13).
package updater
