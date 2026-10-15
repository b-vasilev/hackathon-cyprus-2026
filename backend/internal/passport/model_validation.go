package passport

import (
	"errors"
	"fmt"
	"slices"
)

type modelIndex struct {
	ids, evidence, claims, objects, scenarios map[string]bool
	relations                                 map[string]Relation
}

func (index *modelIndex) register(id string) error {
	if id == "" || index.ids[id] {
		return fmt.Errorf("duplicate or empty object ID: %q", id)
	}
	index.ids[id] = true
	return nil
}

// ValidateModel checks the canonical model independently of any document/views.
func ValidateModel(model Model, documents map[string]int) error {
	_, err := validateModel(model, documents, false)
	return err
}

// ValidateModelDraft checks local fields, IDs and known hierarchy constraints.
// Missing cross-part references and root metadata may be filled by later edits;
// ValidateModel must pass before a draft is finalized.
func ValidateModelDraft(model Model, documents map[string]int) error {
	_, err := validateModel(model, documents, true)
	return err
}

//nolint:gocyclo // Check the canonical entity families against the same ID indexes.
func validateModel(model Model, documents map[string]int, draft bool) (*modelIndex, error) {
	index := &modelIndex{
		ids: make(map[string]bool), evidence: make(map[string]bool), claims: make(map[string]bool),
		objects: make(map[string]bool), scenarios: make(map[string]bool), relations: make(map[string]Relation),
	}
	refs := func(ids []string, known map[string]bool) error {
		if draft {
			return nil
		}
		return checkRefs(ids, known)
	}
	for _, evidence := range model.Evidence {
		if err := index.register(evidence.ID); err != nil {
			return nil, err
		}
		index.evidence[evidence.ID] = true
		lines, known := documents[evidence.DocumentID]
		if !known {
			return nil, fmt.Errorf("evidence %s: unknown document_id %q; use the exact document_id returned by list_sources or read_source", evidence.ID, evidence.DocumentID)
		}
		if evidence.StartLine < 1 || evidence.EndLine < evidence.StartLine || evidence.EndLine > lines {
			return nil, fmt.Errorf("evidence %s: invalid source range %d..%d for document %s; use original lines 1..%d", evidence.ID, evidence.StartLine, evidence.EndLine, evidence.DocumentID, lines)
		}
	}
	for _, claim := range model.Claims {
		if err := index.register(claim.ID); err != nil {
			return nil, err
		}
		index.claims[claim.ID] = true
		if claim.Text == "" {
			return nil, fmt.Errorf("claim %s: text must be nonempty", claim.ID)
		}
		if !slices.Contains([]string{"source_backed", "inferred", "unknown"}, claim.Basis) {
			return nil, fmt.Errorf("claim %s: basis %q must be source_backed, inferred or unknown", claim.ID, claim.Basis)
		}
		if !draft && claim.Basis == "source_backed" && len(claim.EvidenceIDs) == 0 {
			return nil, fmt.Errorf("source-backed claim lacks evidence: %s", claim.ID)
		}
		if err := refs(claim.EvidenceIDs, index.evidence); err != nil {
			return nil, err
		}
	}
	for _, object := range model.Objects {
		if err := index.register(object.ID); err != nil {
			return nil, err
		}
		index.objects[object.ID] = true
		if object.Name == "" {
			return nil, fmt.Errorf("object %s: name must be nonempty", object.ID)
		}
		if !slices.Contains([]string{"repository", "actor", "capability", "entity", "component", "store", "external_system", "interface", "environment", "decision", "owner"}, object.Kind) {
			return nil, fmt.Errorf("object %s: kind %q must be repository, actor, capability, entity, component, store, external_system, interface, environment, decision or owner; software_system/runtime_unit belong in architecture_role with kind component", object.ID, object.Kind)
		}
		if !slices.Contains([]string{"implemented", "in_development", "experimental", "planned", "unknown"}, object.Status) {
			return nil, fmt.Errorf("object %s: status %q must be implemented, in_development, experimental, planned or unknown", object.ID, object.Status)
		}
		if !draft && len(object.ClaimIDs) == 0 {
			return nil, fmt.Errorf("object lacks supporting claims: %s", object.ID)
		}
		if err := refs(object.ClaimIDs, index.claims); err != nil {
			return nil, err
		}
	}
	for _, relation := range model.Relations {
		if err := index.register(relation.ID); err != nil {
			return nil, err
		}
		index.relations[relation.ID] = relation
		if (!draft && (!index.objects[relation.From] || !index.objects[relation.To] || len(relation.ClaimIDs) == 0)) || relation.From == "" || relation.To == "" || !slices.Contains([]string{"call", "data", "dependency", "ownership", "association", "transition"}, relation.Kind) {
			return nil, fmt.Errorf("invalid relation: %s", relation.ID)
		}
		if err := refs(relation.ClaimIDs, index.claims); err != nil {
			return nil, err
		}
		if relation.Interaction != "" && !slices.Contains([]string{"in_process", "sync", "async", "artifact", "build", "unknown"}, relation.Interaction) {
			return nil, fmt.Errorf("invalid relation interaction: %s", relation.ID)
		}
	}
	if err := validateArchitectureHierarchy(model.Objects, draft); err != nil {
		return nil, err
	}
	for _, scenario := range model.Scenarios {
		if err := index.register(scenario.ID); err != nil {
			return nil, err
		}
		index.scenarios[scenario.ID] = true
		for _, step := range scenario.Steps {
			if err := index.register(step.ID); err != nil {
				return nil, err
			}
			if err := refs(step.ObjectIDs, index.objects); err != nil {
				return nil, err
			}
			if err := refs(step.ClaimIDs, index.claims); err != nil {
				return nil, err
			}
		}
	}
	if !draft && (model.Title == "" || model.Summary == "") {
		return nil, errors.New("passport must contain a product title and summary")
	}
	return index, nil
}
