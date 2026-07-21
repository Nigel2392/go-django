package queries

import (
	"context"
	"fmt"
	"strings"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

// scanPlanModelSlot describes a model instance to create when applying
// the scan plan to a new row.
type scanPlanModelSlot struct {
	// creator is the value to pass to attrs.NewObject.
	// nil for the root model (slot 0), which is provided externally.
	creator any

	// parentSlotIdx is the index of the parent model slot.
	// -1 for root or independent models (e.g. through models).
	parentSlotIdx int

	// fkFieldName is the relation field name in the parent model,
	// used for setRelatedObjects when setFK is true.
	fkFieldName string

	// fkRelType is the relation type for the FK setting.
	fkRelType attrs.RelationType

	// setFK indicates whether to call setRelatedObjects on the parent.
	setFK bool
}

// scanPlanEntry describes a single field in the pre-compiled scan plan.
// Entries can be "phantom" (internal chain nodes not included in output)
// or regular (output fields).
type scanPlanEntry struct {
	// isPhantom is true for internal chain node entries that are not output fields.
	isPhantom bool

	// fieldName is used for defs.Field(name) lookup during apply.
	fieldName string

	// slotIdx indexes into scanPlan.modelSlots for the model this field belongs to.
	slotIdx int

	// Metadata copied directly to scannableField:
	relType   attrs.RelationType
	isThrough bool
	chainKey  string
	chainPart string

	// srcEntryIdx is the index in the entries array of the parent scannableField
	// used for srcField linkage. -1 if no parent.
	srcEntryIdx int

	// throughSlotIdx indexes into modelSlots for the associated through model.
	// -1 if no through model is associated.
	throughSlotIdx int

	// virtualField is non-nil for virtual fields that don't belong to any model.
	// Used directly as scannableField.field without model lookup.
	virtualField attrs.Field
}

// scanPlan captures the static structure of [getScannableFields] output.
// It is compiled once per query execution and applied per row to avoid
// repeated schema traversal, string joins, and type assertions.
type scanPlan struct {
	// totalFields is the number of output (non-phantom) entries.
	totalFields int

	// modelSlots describes models to create per row.
	// Slot 0 is always the root model (provided externally).
	modelSlots []scanPlanModelSlot

	// entries includes all plan entries (phantom + output), in order.
	entries []scanPlanEntry

	// rootPrimaryEntryIdx is the index of the first root primary key
	// entry in the entries slice. -1 if no primary key is found.
	rootPrimaryEntryIdx int

	models []attrs.Definer
	defs   []attrs.Definitions
	buf    []scannableField
	out    []*scannableField
}

// compileScanPlan pre-compiles a scan plan from query field info.
// This should be called once per query execution, before the row loop.
// The modelObject parameter is the model prototype (e.g., qs.internals.Model.Object).
func compileScanPlan[T attrs.FieldDefinition](
	ctx context.Context,
	fields []*FieldInfo[T],
	modelObject any,
) *scanPlan {
	sampleRoot := attrs.NewObject[attrs.Definer](ctx, modelObject)

	plan := &scanPlan{
		rootPrimaryEntryIdx: -1,
		modelSlots: []scanPlanModelSlot{{
			parentSlotIdx: -1,
		}},
	}

	chainKeyToSlotIdx := make(map[string]int)
	chainKeyToPhantomIdx := make(map[string]int)

	for _, info := range fields {
		// --- Through model ---
		var throughSlotIdx = -1
		if info.Through != nil {
			throughSlotIdx = len(plan.modelSlots)
			plan.modelSlots = append(plan.modelSlots, scanPlanModelSlot{
				creator:       info.Through.Model,
				parentSlotIdx: -1,
			})
			for _, f := range info.Through.Fields {
				plan.entries = append(plan.entries, scanPlanEntry{
					fieldName:      f.Name(),
					slotIdx:        throughSlotIdx,
					relType:        info.RelType,
					isThrough:      true,
					srcEntryIdx:    -1,
					throughSlotIdx: -1,
				})
			}
		}

		// --- Root fields (SourceField is zero value) ---
		if any(info.SourceField) == any(*(new(T))) {
			rootDefs := attrs.Define(ctx, sampleRoot)
			for _, f := range info.Fields {
				// Virtual field check
				if virt, ok := any(f).(VirtualField); ok && info.Model == nil {
					attrField, ok := virt.(attrs.Field)
					if !ok {
						panic(fmt.Errorf("virtual field %q does not implement attrs.Field", f.Name()))
					}
					plan.entries = append(plan.entries, scanPlanEntry{
						slotIdx:        0,
						relType:        -1,
						srcEntryIdx:    -1,
						throughSlotIdx: throughSlotIdx,
						virtualField:   attrField,
					})
					continue
				}

				field, ok := rootDefs.Field(f.Name())
				if !ok {
					panic(fmt.Errorf("field %q not found in %T", f.Name(), sampleRoot))
				}

				if field.IsPrimary() && plan.rootPrimaryEntryIdx == -1 {
					plan.rootPrimaryEntryIdx = len(plan.entries)
				}

				plan.entries = append(plan.entries, scanPlanEntry{
					fieldName:      f.Name(),
					slotIdx:        0,
					relType:        -1,
					srcEntryIdx:    -1,
					throughSlotIdx: throughSlotIdx,
				})
			}
			continue
		}

		// --- Chained fields ---

		// Walk chain to create model slots and phantom entries
		for i, name := range info.Chain {
			key := strings.Join(info.Chain[:i+1], ".")
			if _, exists := chainKeyToSlotIdx[key]; exists {
				continue
			}

			var parentSlotIdx = 0
			var parentKey string
			if i > 0 {
				parentKey = strings.Join(info.Chain[:i], ".")
				parentSlotIdx = chainKeyToSlotIdx[parentKey]
			}

			// Inspect parent model for relation info
			var parentModel any
			if parentSlotIdx == 0 {
				parentModel = modelObject
			} else {
				parentModel = plan.modelSlots[parentSlotIdx].creator
			}
			parentObj := attrs.NewObject[attrs.Definer](ctx, parentModel)
			parentDefs := attrs.Define(ctx, parentObj)
			relField, ok := parentDefs.Field(name)
			if !ok {
				panic(fmt.Errorf("field %q not found in %T", name, parentObj))
			}

			rel := relField.Rel()
			relType := rel.Type()

			var modelCreator any
			if i == len(info.Chain)-1 {
				modelCreator = info.Model
			} else {
				modelCreator = rel.Model()
			}

			slotIdx := len(plan.modelSlots)
			plan.modelSlots = append(plan.modelSlots, scanPlanModelSlot{
				creator:       modelCreator,
				parentSlotIdx: parentSlotIdx,
				fkFieldName:   name,
				fkRelType:     relType,
				setFK:         relType == attrs.RelManyToOne,
			})
			chainKeyToSlotIdx[key] = slotIdx

			// Determine srcEntryIdx for this chain node
			srcEntryIdx := -1
			if parentKey == "" {
				srcEntryIdx = plan.rootPrimaryEntryIdx
			} else {
				srcEntryIdx = chainKeyToPhantomIdx[parentKey]
			}

			chainKeyToPhantomIdx[key] = len(plan.entries)
			plan.entries = append(plan.entries, scanPlanEntry{
				isPhantom:      true,
				slotIdx:        slotIdx,
				relType:        relType,
				chainKey:       key,
				chainPart:      name,
				srcEntryIdx:    srcEntryIdx,
				throughSlotIdx: -1,
			})
		}

		// Add leaf field entries
		lastKey := strings.Join(info.Chain, ".")
		chainSlotIdx := chainKeyToSlotIdx[lastKey]
		phantom := &plan.entries[chainKeyToPhantomIdx[lastKey]]

		for _, f := range info.Fields {
			plan.entries = append(plan.entries, scanPlanEntry{
				fieldName:      f.Name(),
				slotIdx:        chainSlotIdx,
				relType:        phantom.relType,
				chainKey:       phantom.chainKey,
				chainPart:      phantom.chainPart,
				srcEntryIdx:    phantom.srcEntryIdx,
				throughSlotIdx: throughSlotIdx,
			})
		}
	}

	// Count output fields
	for _, e := range plan.entries {
		if !e.isPhantom {
			plan.totalFields++
		}
	}

	plan.models = make([]attrs.Definer, len(plan.modelSlots))
	plan.defs = make([]attrs.Definitions, len(plan.modelSlots))
	plan.buf = make([]scannableField, len(plan.entries))
	plan.out = make([]*scannableField, plan.totalFields)

	return plan
}

// apply executes the scan plan for a single row, creating model instances,
// resolving fields, and returning the scannableField slice ready for
// value scanning. The root model is provided externally.
func (plan *scanPlan) apply(
	ctx context.Context,
	root attrs.Definer,
) {
	// Create model instances for each slot
	plan.models[0] = root

	for i := 1; i < len(plan.modelSlots); i++ {
		slot := &plan.modelSlots[i]
		obj := attrs.NewObject[attrs.Definer](ctx, slot.creator)
		plan.models[i] = obj
		if slot.setFK {
			setRelatedObjects(
				ctx, slot.fkFieldName, slot.fkRelType,
				plan.models[slot.parentSlotIdx],
				[]Relation{&baseRelation{object: obj}},
			)
		}
	}

	// Get definitions for each model (one Define call per unique model)
	for i, model := range plan.models {
		if model != nil {
			plan.defs[i] = attrs.Define(ctx, model)
		}
	}

	// Build scannableField entries using a contiguous backing array
	outputIdx := 0

	for i := range plan.entries {
		e := &plan.entries[i]
		sf := &plan.buf[i]

		sf.relType = e.relType
		sf.isThrough = e.isThrough
		sf.chainKey = e.chainKey
		sf.chainPart = e.chainPart

		if e.srcEntryIdx >= 0 {
			sf.srcField = &plan.buf[e.srcEntryIdx]
		}

		if e.throughSlotIdx >= 0 {
			sf.through = plan.models[e.throughSlotIdx]
		}

		// Virtual field: use directly, no model object
		if e.virtualField != nil {
			sf.field = e.virtualField
			sf.idx = outputIdx
			plan.out[outputIdx] = sf
			outputIdx++
			continue
		}

		// Set object for non-virtual entries
		sf.object = plan.models[e.slotIdx]

		// Phantom entry: use primary field, not part of output
		if e.isPhantom {
			sf.field = plan.defs[e.slotIdx].Primary()
			sf.idx = -1
			continue
		}

		// Regular field: look up by name
		field, ok := plan.defs[e.slotIdx].Field(e.fieldName)
		if !ok {
			panic(fmt.Errorf(
				"field %q not found in %T during scan plan apply",
				e.fieldName, plan.models[e.slotIdx],
			))
		}
		sf.field = field
		sf.idx = outputIdx
		plan.out[outputIdx] = sf
		outputIdx++
	}
}
