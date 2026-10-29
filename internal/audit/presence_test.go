//go:build integration

package audit

// nullableValue keeps the old test dereference behavior: an absent/null value
// fails the test instead of silently turning into the element's zero value.
func nullableValue[T any](n interface{ Get() (T, error) }) T {
	value, err := n.Get()
	if err != nil {
		panic(err)
	}
	return value
}
