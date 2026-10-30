# Empty-profile setup diagnostic

The initial fixture used an empty DNS profile. Publication remained blocked
with `dns_policy` failed / `VALIDATION_FAILED` and “Resolver policy was not
applied.” No product traffic acceptance was claimed. The final fixture uses
the members harness's nonempty controlled allow rule. This is retained as setup
evidence, not filed as an isolation or node enforcement defect.
