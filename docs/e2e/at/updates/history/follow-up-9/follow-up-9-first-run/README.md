# Follow-up 9 diagnostic run

Source `e112595` on product main `5575470587a4e3014bd515e624ba2ad0aacbd140`. AT23 failed health because the agent used the devstack random boot UUID and the updater used the kernel boot ID. Later regular scenarios were blocked by missing last-good. The installer ran after a fleet default existed, so its fresh-node missing-desired-state prerequisite was not met. No pass claims from this run are carried into the current run.

The final harness aligns the regular lab boot input before enrollment and measures the installer before any fleet default, with the first agent poll held by a runtime systemd condition.
