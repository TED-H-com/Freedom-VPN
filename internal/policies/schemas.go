package policies

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schemas/*.schema.json
var schemasFS embed.FS

// SchemasFS returns the embedded JSON Schema 2020-12 files for all six policy
// payload types.
func SchemasFS() embed.FS {
	return schemasFS
}

var schemaFiles = map[api.PolicyType]string{
	api.PolicyTypePlan:           "schemas/plan-payload.schema.json",
	api.PolicyTypeFairUse:        "schemas/fair-use-payload.schema.json",
	api.PolicyTypeDnsProfile:     "schemas/dns-profile-payload.schema.json",
	api.PolicyTypeDomainRuleSet:  "schemas/domain-rule-set-payload.schema.json",
	api.PolicyTypeThreatFeed:     "schemas/threat-feed-payload.schema.json",
	api.PolicyTypeReleaseSupport: "schemas/release-support-payload.schema.json",
}

var (
	compiledSchemasOnce sync.Once
	compiledSchemas     map[api.PolicyType]*jsonschema.Schema
	compileSchemasErr   error
)

func getCompiledSchemas() (map[api.PolicyType]*jsonschema.Schema, error) {
	compiledSchemasOnce.Do(func() {
		compiledSchemas = make(map[api.PolicyType]*jsonschema.Schema, len(schemaFiles))
		for pt, path := range schemaFiles {
			raw, err := schemasFS.ReadFile(path)
			if err != nil {
				compileSchemasErr = fmt.Errorf("read embedded schema %s: %w", path, err)
				return
			}
			doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
			if err != nil {
				compileSchemasErr = fmt.Errorf("unmarshal embedded schema %s: %w", path, err)
				return
			}
			c := jsonschema.NewCompiler()
			c.AssertFormat()
			url := "https://freedom-vpn.example.invalid/" + path
			if err := c.AddResource(url, doc); err != nil {
				compileSchemasErr = fmt.Errorf("add schema resource %s: %w", path, err)
				return
			}
			sch, err := c.Compile(url)
			if err != nil {
				compileSchemasErr = fmt.Errorf("compile schema %s: %w", path, err)
				return
			}
			compiledSchemas[pt] = sch
		}
	})
	return compiledSchemas, compileSchemasErr
}

// ValidateJSONSchema validates rawPayload against the versioned JSON Schema
// 2020-12 for policyType.
func ValidateJSONSchema(policyType api.PolicyType, rawPayload []byte) error {
	schemas, err := getCompiledSchemas()
	if err != nil {
		return err
	}
	sch, ok := schemas[policyType]
	if !ok {
		return newValidationError("/type", ReasonUnknownPolicyType, fmt.Sprintf("unsupported policy type %q", policyType))
	}

	var rawObj map[string]any
	if err := json.Unmarshal(rawPayload, &rawObj); err != nil {
		return newValidationError("/payload", ReasonCanonicalJSONInvalid, "policy payload must be a JSON object")
	}

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(rawPayload))
	if err != nil {
		return newValidationError("/payload", ReasonCanonicalJSONInvalid, "policy payload is not valid JSON")
	}
	if err := sch.Validate(inst); err != nil {
		return classifySchemaError(policyType, rawObj, err)
	}
	return nil
}

// classifySchemaError maps structural schema failures onto fine-grained
// RuleReason values when a semantic rule is also constrained by the schema,
// so callers testing specific failure cases receive deterministic reasons.
func classifySchemaError(policyType api.PolicyType, rawObj map[string]any, schErr error) error {
	switch policyType {
	case api.PolicyTypeFairUse:
		if rawObj != nil {
			tiersRaw, hasTiers := rawObj["tiers"]
			if !hasTiers || tiersRaw == nil {
				return newValidationError("/payload/tiers", ReasonFairUseEmptyTiers, "fair_use policy must define at least one tier")
			}
			if tiersAny, ok := tiersRaw.([]any); ok {
				if len(tiersAny) == 0 {
					return newValidationError("/payload/tiers", ReasonFairUseEmptyTiers, "fair_use policy must define at least one tier")
				}
				for i, tAny := range tiersAny {
					if tMap, ok := tAny.(map[string]any); ok {
						_, hasDown := tMap["down_bps"]
						_, hasUp := tMap["up_bps"]
						if !hasDown || !hasUp {
							return newValidationError(fmt.Sprintf("/payload/tiers/%d", i), ReasonFairUseMissingDirection, "every fair_use tier must specify both down_bps and up_bps")
						}
						if downF, ok := tMap["down_bps"].(float64); ok && downF < 0 {
							return newValidationError(fmt.Sprintf("/payload/tiers/%d/down_bps", i), ReasonFairUseNegativeRate, "tier down_bps must be non-negative")
						}
						if upF, ok := tMap["up_bps"].(float64); ok && upF < 0 {
							return newValidationError(fmt.Sprintf("/payload/tiers/%d/up_bps", i), ReasonFairUseNegativeRate, "tier up_bps must be non-negative")
						}
						if prioF, ok := tMap["priority"].(float64); ok && (prioF < 0 || prioF > 7) {
							return newValidationError(fmt.Sprintf("/payload/tiers/%d/priority", i), ReasonFairUsePriorityBounds, "tier priority must be between 0 and 7")
						}
					}
				}
			}
			if minAny, ok := rawObj["minimum_rate"].(map[string]any); ok {
				_, hasDown := minAny["down_bps"]
				_, hasUp := minAny["up_bps"]
				if !hasDown || !hasUp {
					return newValidationError("/payload/minimum_rate", ReasonFairUseMissingDirection, "minimum_rate must specify both down_bps and up_bps")
				}
				if downF, ok := minAny["down_bps"].(float64); ok && downF < 0 {
					return newValidationError("/payload/minimum_rate/down_bps", ReasonFairUseNegativeRate, "minimum_rate down_bps must be non-negative")
				}
				if upF, ok := minAny["up_bps"].(float64); ok && upF < 0 {
					return newValidationError("/payload/minimum_rate/up_bps", ReasonFairUseNegativeRate, "minimum_rate up_bps must be non-negative")
				}
			}
			if ovsAny, ok := rawObj["overrides"].([]any); ok {
				for i, oAny := range ovsAny {
					if oMap, ok := oAny.(map[string]any); ok {
						_, hasDown := oMap["down_bps"]
						_, hasUp := oMap["up_bps"]
						if !hasDown || !hasUp {
							return newValidationError(fmt.Sprintf("/payload/overrides/%d", i), ReasonFairUseMissingDirection, "override must specify both down_bps and up_bps")
						}
						if downF, ok := oMap["down_bps"].(float64); ok && downF < 0 {
							return newValidationError(fmt.Sprintf("/payload/overrides/%d/down_bps", i), ReasonFairUseNegativeRate, "override down_bps must be non-negative")
						}
						if upF, ok := oMap["up_bps"].(float64); ok && upF < 0 {
							return newValidationError(fmt.Sprintf("/payload/overrides/%d/up_bps", i), ReasonFairUseNegativeRate, "override up_bps must be non-negative")
						}
					}
				}
			}
		}
	case api.PolicyTypeDomainRuleSet:
		if rawObj != nil {
			if rulesAny, ok := rawObj["rules"].([]any); ok {
				for i, rAny := range rulesAny {
					if rMap, ok := rAny.(map[string]any); ok {
						if dom, ok := rMap["domain"].(string); ok {
							if err := checkDomainIsolationBaseline(dom, fmt.Sprintf("/payload/rules/%d/domain", i)); err != nil {
								return err
							}
							norm, err := dnspolicy.NormalizeDomain(dom)
							if err != nil || norm.Canonical != dom {
								return newValidationError(fmt.Sprintf("/payload/rules/%d/domain", i), ReasonDNSDomainNotCanonical, fmt.Sprintf("domain %q is not in canonical IDNA2008 ASCII form", dom))
							}
						}
					}
				}
			}
		}
	case api.PolicyTypeReleaseSupport:
		if rawObj != nil {
			if msv, ok := rawObj["minimum_serving_version"].(map[string]any); ok {
				if serving, ok := msv["serving_release_ids"].([]any); ok && len(serving) == 0 {
					return newValidationError("/payload/minimum_serving_version/serving_release_ids", ReasonReleaseServingEmpty, "minimum_serving_version.serving_release_ids must not be empty")
				}
			}
		}
	case api.PolicyTypeThreatFeed:
		if rawObj != nil {
			if srcURL, ok := rawObj["source_url"].(string); ok && !strings.HasPrefix(srcURL, "https://") {
				return newValidationError("/payload/source_url", ReasonThreatFeedInvalidURL, "source_url must use https://")
			}
			if licURL, ok := rawObj["license_url"].(string); ok && !strings.HasPrefix(licURL, "https://") {
				return newValidationError("/payload/license_url", ReasonThreatFeedInvalidURL, "license_url must use https://")
			}
		}
	}
	return newValidationError("/payload", ReasonSchemaInvalid, fmt.Sprintf("policy payload failed JSON schema validation: %v", schErr))
}
