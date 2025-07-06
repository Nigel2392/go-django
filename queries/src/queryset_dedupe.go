package queries

import (
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/elliotchance/orderedmap/v2"
)

// objectRelation contains metadata about the list of related objects and
// the relation type itself.
type objectRelation struct {
	relTyp  attrs.RelationType
	objects *orderedmap.OrderedMap[any, *object]
}

// An object is a representation of a model instance in the rows structure.
//
// It contains the primary key, the field definitions, and the relations of the object.
//
// Any relations stored on this object are directly related to the object itself,
// if a through model is used, it is stored in the `through` field.
type object struct {
	// through is a possible through model for the relation
	through attrs.Definer

	// the primary key of the object
	uniqueValue any

	// the field defs of the object
	fieldDefs attrs.Definitions

	// The object itself, which is a Definer
	obj attrs.Definer

	// the direct relations of the object (multiple)
	relations map[string]*objectRelation
}

// the rootObject provides the top layer of the [rows] structure.
//
// It contains the object itself, and any annotations that are associated with it.
type rootObject struct {
	object      *object
	annotations map[string]any // Annotations for the root object
}

// rows represents a collection of root objects.
//
// each of those root objects can have multiple relations to other objects,
// which are stored in the [object] struct.
//
// The rows structure is used to deduplicate relations and compile the final result set.
//
// for deduplication of multi- valued relations, the primary key of the parent and child objects
// have to be set, otherwise the relation cannot be deduplicated.
type rows[T attrs.Definer] struct {
	anyOfRootScannable *scannableField   // any scannable field that can be used to retrieve the root object
	possibleDuplicates []*scannableField // possible duplicate fields that can be added to the rows
	hasMultiRelations  bool              // if the rows have multi-valued relations

	objects *orderedmap.OrderedMap[any, *rootObject]
	forEach func(attrs.Definer) error
}

// Initialize a new rows structure for the given model type.
// It will scan the fields of the model and build a list of scannable fields that can be used to retrieve the root object.
// It will also add possible duplicate fields to the list, which can be used to deduplicate relations later on.
// The forEach function is called for each object that is added to the rows structure,
func newRows[T attrs.Definer](fields []*FieldInfo[attrs.FieldDefinition], mdl attrs.Definer, forEach func(attrs.Definer) error) (*rows[T], error) {
	var seen = make(map[string]struct{}, 0)
	var scannables = getScannableFields(
		fields, mdl,
	)

	var r = &rows[T]{
		objects:            orderedmap.NewOrderedMap[any, *rootObject](),
		possibleDuplicates: make([]*scannableField, 0),
		hasMultiRelations:  false,
		forEach:            forEach,
	}

	// add possible duplicate fields to the list
	//
	// also add o2o relations, this will
	// make sure the through model gets set later on
	for _, scannable := range scannables {
		// check if field is a multi-valued relation
		if (scannable.relType == attrs.RelManyToMany ||
			scannable.relType == attrs.RelOneToMany ||
			scannable.relType == attrs.RelOneToOne) &&
			// check if primary and not through object
			!scannable.isThrough {

			if scannable.relType != attrs.RelOneToOne {
				r.hasMultiRelations = true
			}

			if _, ok := seen[scannable.chainKey]; ok {
				continue
			}

			r.possibleDuplicates = append(r.possibleDuplicates, scannable)
		}

		if scannable.relType == -1 && scannable.object != nil && scannable.field.IsPrimary() {
			r.anyOfRootScannable = scannable
		}

		if scannable.relType == -1 && scannable.object != nil && r.anyOfRootScannable == nil {
			r.anyOfRootScannable = scannable
		}
	}

	if r.hasMultiRelations && r.anyOfRootScannable == nil {
		return nil, fmt.Errorf(
			"no root scannable field found for model %T, cannot build relations", mdl,
		)
	}

	return r, nil
}

// hasRoot checks if the rows structure has a root object.
// it returns true if there is a scannable field that can be used to retrieve the root object,
// otherwise it returns false.
func (r *rows[T]) hasRoot() bool {
	// if the root scannable has no unique value, it is not a root object
	return r.anyOfRootScannable != nil
}

// rootRow returns the root row of the rows structure.
// it returns the scannable field (from the list provided)
// that can be used to retrieve the root object
func (r *rows[T]) rootRow(scannables []*scannableField) *scannableField {
	if r.anyOfRootScannable != nil {
		return scannables[r.anyOfRootScannable.idx]
	}
	return nil
}

// addRoot adds a root object to the rows structure.
//
// this is used to add the top-level object to the rows,
// which can then have relations added to it.
//
// it has to be called before any relations are added - technically
// root objects can be added inside of the [addRelationChain] method,
// but this would lose any annotations that are associated with the root object.
func (r *rows[T]) addRoot(uniqueValue any, obj attrs.Definer, through attrs.Definer, annotations map[string]any) *rootObject {
	if uniqueValue == nil {
		panic("cannot add root object with nil primary key")
	}

	if root, ok := r.objects.Get(uniqueValue); ok {
		return root
	}

	var defs attrs.Definitions
	if obj != nil {
		defs = obj.FieldDefs()
	}

	var root = &rootObject{
		object: &object{
			uniqueValue: uniqueValue,
			obj:         obj,
			fieldDefs:   defs,
			through:     through,
			relations:   make(map[string]*objectRelation),
		},
		annotations: annotations,
	}

	r.objects.Set(uniqueValue, root)
	return root
}

// addRelationChain adds a relation chain to the rows structure.
//
// it is used to add relations to the root object, or any other object in the rows structure.
// the chain is a list of [chainPart] that contains the relation type, primary key, and the object itself.
//
// the root object has to be added with [addRoot] before this method is called,
// otherwise it will panic.
func (r *rows[T]) addRelationChain(chain []chainPart) {

	var root = chain[0]
	var obj, ok = r.objects.Get(root.uniqueValue)
	if !ok {
		panic(fmt.Sprintf("root object with primary key %v not found in rows, root needs to be added with rows.addRoot", root.uniqueValue))
	}
	var current = obj.object
	var idx = 1
	for idx < len(chain) {
		var part = chain[idx]

		// If the primary key is zero and the relation is not a ManyToOne or OneToOne,
		// we can stop traversing the chain, as there is no data for this relation
		//
		// This is to exclude empty rows in the result set when querying multiple- valued relations.
		//
		// ManyToOne and OneToOne relations are special cases where the primary key can be zero.
		//
		// This also means that any deeper relations cannot be traversed, I.E. we break the loop.
		if fields.IsZero(part.uniqueValue) && !(part.relTyp == attrs.RelManyToOne || part.relTyp == attrs.RelOneToOne) {
			break
		}

		var next, ok = current.relations[part.chain]
		if !ok {
			next = &objectRelation{
				relTyp:  part.relTyp,
				objects: orderedmap.NewOrderedMap[any, *object](),
			}
			current.relations[part.chain] = next
		}

		child, ok := next.objects.Get(part.uniqueValue)
		if !ok {
			// child does not exist, create and add it
			var through attrs.Definer
			if part.through != nil {
				// If there is a through object, we need to set it
				through = part.through
			}

			child = &object{
				uniqueValue: part.uniqueValue,
				fieldDefs:   part.object.FieldDefs(),
				obj:         part.object,
				relations:   make(map[string]*objectRelation),
				through:     through,
			}

			next.objects.Set(part.uniqueValue, child)
		}

		current = child
		idx++
	}
}

func (r *rows[T]) compile(qs *QuerySet[T]) (Rows[T], error) {
	var addRelations func(*object, uint64) error
	// addRelations is a recursive function that traverses the object and its relations,
	// and sets the related objects on the provided parent object.
	addRelations = func(obj *object, depth uint64) error {

		if obj.uniqueValue == nil {
			panic(fmt.Sprintf("object %T has no primary key, cannot deduplicate relations", obj.obj))
		}

		for relName, rel := range obj.relations {
			var relatedObjects = make([]Relation, 0, rel.objects.Len())
			for relHead := rel.objects.Front(); relHead != nil; relHead = relHead.Next() {
				var relatedObj = relHead.Value
				if relatedObj == nil {
					continue
				}

				if err := addRelations(relatedObj, depth+1); err != nil {
					return fmt.Errorf("object %T: %w", obj, err)
				}

				var throughObj attrs.Definer
				if relatedObj.through != nil {
					// If there is a through object, we need to add it to the related objects
					throughObj = relatedObj.through

					// If the related object implements ThroughModelSetter, we set the through model
					// directly on the object here as opposed to in the [setRelatedObjects] function.
					// This is to avoid complex and unreadable code in the [setRelatedObjects] switch case.
					if def, ok := relatedObj.obj.(ThroughModelSetter); ok {
						def.SetThroughModel(throughObj)
					}
				}

				relatedObjects = append(relatedObjects, &baseRelation{
					uniqueValue: relatedObj.uniqueValue,
					object:      relatedObj.obj,
					through:     throughObj,
				})
			}

			// aways set the related objects on the parent object
			setRelatedObjects(relName, rel.relTyp, obj.obj, relatedObjects)
		}

		if r.forEach != nil {
			if obj.through != nil {
				// Call the forEach function with the through object if it exists
				if err := r.forEach(obj.through); err != nil {
					return fmt.Errorf("error in forEach[%d] for through object %T: %w", depth, obj.through, err)
				}
			}

			// If a forEach function is set, we call it for each object
			if err := r.forEach(obj.obj); err != nil {
				return fmt.Errorf("error in forEach[%d] for object %T: %w", depth, obj.obj, err)
			}
		}

		return nil
	}

	var root = make([]*Row[T], 0, r.objects.Len())
	for head := r.objects.Front(); head != nil; head = head.Next() {
		var obj = head.Value
		if obj == nil {
			continue
		}

		// add the relations to each object recursively
		if err := addRelations(obj.object, 0); err != nil {
			return nil, fmt.Errorf("failed to add relations for object with primary key %v: %w", obj.object.uniqueValue, err)
		}

		var definer = obj.object.obj
		if definer == nil {
			continue
		}

		// Annotate the object if it implements the Annotator interface
		if annotator, ok := definer.(Annotator); ok {
			annotator.Annotate(obj.annotations)
		}

		// If the definer implements ThroughModelSetter, we set the through model directly on the object.
		// This is not done inside the [setRelatedObjects] function to avoid
		// unreadable and complex code.
		if throughSetter, ok := definer.(ThroughModelSetter); ok && obj.object.through != nil {
			throughSetter.SetThroughModel(obj.object.through)
		}

		root = append(root, &Row[T]{
			Through:     obj.object.through,
			Object:      definer.(T),
			Annotations: obj.annotations,
		})
	}

	return root, nil
}

// a chainPart represents a part of a relation chain.
// it contains information about the relation and object.
type chainPart struct {
	relTyp      attrs.RelationType
	chain       string
	uniqueValue any
	object      attrs.Definer
	through     attrs.Definer
}

// buildChainParts builds a chain of parts from the actual field to the parent field.
//
// It traverses the scannableField structure and collects the relation type, primary key,
// object, and through model for each part of the chain.
//
// The [getScannableFields] function builds this chain of *scannableField objects,
// which represent the fields that can be scanned from the database.
func buildChainParts(actualField *scannableField) []chainPart {
	// Get the stack of fields from target to parent
	var stack = make([]chainPart, 0)
	for cur := actualField; cur != nil; cur = cur.srcField {
		var (
			inst    = cur.object
			defs    = inst.FieldDefs()
			primary = defs.Primary()
		)

		var (
			pk         = primary.GetValue()
			primaryVal = pk
		)
		if primaryVal == nil || fields.IsZero(primaryVal) {
			var err error
			pk, err = GetUniqueKey(defs)
			if err != nil && !errors.Is(err, errors.NoUniqueKey) {
				panic(fmt.Sprintf("error getting unique key for field %s: %v", cur.chainKey, err))
			}
		}

		if pk == nil && primaryVal != nil {
			pk = primaryVal
		}

		if (cur.relType == attrs.RelManyToMany || cur.relType == attrs.RelOneToMany) && fields.IsZero(pk) {
			panic(fmt.Sprintf(
				"cannot build chain part for field %s with zero primary key in ManyToMany or OneToMany relation", cur.chainKey,
			))
		}

		stack = append(stack, chainPart{
			relTyp:      cur.relType,
			chain:       cur.chainPart,
			uniqueValue: pk,
			object:      inst,
			through:     cur.through,
		})
	}

	// Reverse the stack to get the fields in the correct order
	// i.e. parent to target
	for i, j := 0, len(stack)-1; i < j; i, j = i+1, j-1 {
		stack[i], stack[j] = stack[j], stack[i]
	}

	return stack
}
