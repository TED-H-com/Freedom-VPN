package config

// SettingError identifies a failed setting without making its value loggable.
// Entrypoints may log Setting(), but must not log arbitrary wrapped errors.
type SettingError struct {
	Name string
	Err  error
}

func (e *SettingError) Error() string   { return e.Name + ": " + e.Err.Error() }
func (e *SettingError) Unwrap() error   { return e.Err }
func (e *SettingError) Setting() string { return e.Name }
