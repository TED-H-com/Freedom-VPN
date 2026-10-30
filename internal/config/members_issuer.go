package config

import "github.com/TED-H-com/Freedom-VPN/internal/members/issuer"

// LoadMembersForIssuer validates the existing central issuer, not a second
// setting in the member file. Both binary domain builders use this loader.
func LoadMembersForIssuer(path, oidcIssuer string) (Members, error) {
	c, err := LoadMembers(path)
	if err != nil {
		return c, &SettingError{Name: "members_config_file", Err: err}
	}
	if err := issuer.Validate(oidcIssuer); err != nil {
		return Members{}, &SettingError{Name: "oidc_issuer", Err: err}
	}

	return c, nil
}
