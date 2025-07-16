package queries

import (
	"fmt"
	"iter"

	"github.com/Nigel2392/go-django/queries/internal"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
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

	seen    map[string]*seenObject // seen is used to deduplicate relations
	objects *orderedmap.OrderedMap[any, *rootObject]
	forEach func(attrs.Definer) error
	qs      *QuerySet[T]
}

type seenObject struct {
	pks     []any
	objects map[any]*object
}

// Initialize a new rows structure for the given model type.
// It will scan the fields of the model and build a list of scannable fields that can be used to retrieve the root object.
// It will also add possible duplicate fields to the list, which can be used to deduplicate relations later on.
// The forEach function is called for each object that is added to the rows structure,
func newRows[T attrs.Definer](qs *QuerySet[T], forEach func(attrs.Definer) error) (*rows[T], error) {
	var seen = make(map[string]struct{}, 0)
	var fields = qs.internals.Fields
	var model = internal.NewObjectFromIface(qs.internals.Model.Object)
	var scannables = getScannableFields(
		fields, model,
	)

	var r = &rows[T]{
		objects:            orderedmap.NewOrderedMap[any, *rootObject](),
		possibleDuplicates: make([]*scannableField, 0),
		hasMultiRelations:  false,
		forEach:            forEach,
		seen:               make(map[string]*seenObject, 0),
		qs:                 qs,
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
			"no root scannable field found for model %T, cannot build relations", model,
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

	var seenM, ok = r.seen[""]
	if !ok {
		seenM = &seenObject{
			pks:     make([]any, 0, 1),
			objects: make(map[any]*object, 0),
		}
		r.seen[""] = seenM
	}

	seenM.pks = append(seenM.pks, uniqueValue)
	seenM.objects[uniqueValue] = root.object

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
func (r *rows[T]) addRelationChain(scannable *scannableField) {

	var chain = r.buildChainParts(scannable)
	var root = chain[0]
	var obj, ok = r.objects.Get(root.uniqueValue)
	if !ok {
		panic(fmt.Sprintf("root object with primary key %v %T not found in rows, root needs to be added with rows.addRoot", root.uniqueValue, root.uniqueValue))
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
		var isZeroVal = fields.IsZero(part.uniqueValue)
		if isZeroVal && !(part.relTyp == attrs.RelManyToOne || part.relTyp == attrs.RelOneToOne) {
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

			// Store seen objects in map - this is later used for prefetching relations
			// and deduplicating relations.
			var seenM, ok = r.seen[part.chain]
			if !ok {
				seenM = &seenObject{
					pks:     make([]any, 0, 1),
					objects: make(map[any]*object, 0),
				}
				r.seen[part.chain] = seenM
			}

			seenM.pks = append(seenM.pks, part.uniqueValue)
			seenM.objects[part.uniqueValue] = child

			next.objects.Set(part.uniqueValue, child)
		}

		current = child
		idx++
	}
}

func (r *rows[T]) queryPreloads(preload *Preload) error {
	if len(r.seen) == 0 {
		return errors.NoUniqueKey.Wrapf(
			"QuerySet.All: no 'seen' map for preload %q", preload.FieldName,
		)
	}

	var seenObj = r.seen[preload.ParentPath]
	if seenObj == nil {
		return errors.ValueError.WithCause(fmt.Errorf(
			"QuerySet.All: no primary key map for preload %q / %q in %v", preload.FieldName, preload.ParentPath, r.seen,
		))
	}

	if len(seenObj.pks) == 0 {
		// cannot query this preload, no primary keys found
		// return to avoid unnecessary queries
		return nil
	}

	var (
		// relType    = preload.Rel.Type()
		relThrough = preload.Rel.Through()
	)

	var subQueryset = preload.QuerySet
	if subQueryset == nil {
		subQueryset = GetQuerySet(preload.Model)
	}

	subQueryset = subQueryset.WithContext(r.qs.Context())

	var targetFieldInfo = &FieldInfo[attrs.FieldDefinition]{
		Model: subQueryset.internals.Model.Object,
		Table: Table{
			Name: subQueryset.internals.Model.Table,
		},
		Fields: ForSelectAllFields[attrs.FieldDefinition](
			subQueryset.internals.Model.Fields,
		),
	}

	if relThrough != nil {
		var throughObject = newThroughProxy(relThrough)

		targetFieldInfo.Through = &FieldInfo[attrs.FieldDefinition]{
			Model: throughObject.object,
			Table: Table{
				Name: throughObject.defs.TableName(),
				Alias: fmt.Sprintf(
					"%s_through",
					subQueryset.internals.Model.Table,
				),
			},
			Fields: ForSelectAllFields[attrs.FieldDefinition](throughObject.defs),
		}

		var condition = &JoinDefCondition{
			Operator: expr.EQ,
			ConditionA: expr.TableColumn{
				TableOrAlias: targetFieldInfo.Table.Name,
				FieldColumn:  r.qs.internals.Model.Primary,
			},
			ConditionB: expr.TableColumn{
				TableOrAlias: targetFieldInfo.Through.Table.Alias,
				FieldColumn:  throughObject.targetField,
			},
		}

		condition.Next = &JoinDefCondition{
			Operator: expr.IN,
			ConditionA: expr.TableColumn{
				TableOrAlias: targetFieldInfo.Through.Table.Alias,
				FieldColumn:  throughObject.sourceField,
			},
			ConditionB: expr.TableColumn{
				Values: []any{seenObj.pks},
			},
		}

		var join = JoinDef{
			TypeJoin: TypeJoinInner,
			Table: Table{
				Name: throughObject.defs.TableName(),
				Alias: fmt.Sprintf(
					"%s_through",
					subQueryset.internals.Model.Table,
				),
			},
			JoinDefCondition: condition,
		}

		subQueryset.internals.AddJoin(join)
	} else {
		var pks = seenObj.pks
		var targetField = preload.Rel.Field()
		if targetField == nil {
			targetField = subQueryset.internals.Model.Primary
		}

		// If the relation is a forward o2o or m2o relation,
		// the PK list is not actually the primary keys of the related objects,
		// but the primary keys of the parent objects - change the pk list accordingly.
		if preload.Rel.Type() == attrs.RelManyToOne || preload.Rel.Type() == attrs.RelOneToOne && preload.Rel.Through() == nil {
			pks = make([]any, 0, len(seenObj.pks))
			for _, obj := range seenObj.objects {
				var field, ok = obj.fieldDefs.Field(preload.FieldName)
				if !ok {
					return errors.FieldNotFound.Wrapf(
						"QuerySet.All: preload %q has no field %q in model %T (%#v)",
						preload.FieldName, preload.Field.Name(), r.qs.internals.Model.Object, preload,
					)
				}

				var val, err = field.Value()
				if err != nil {
					return errors.Wrapf(
						err, "failed to get value for field %q (%#v)", field.Name(), preload,
					)
				}

				if !fields.IsZero(val) {
					pks = append(pks, val)
				}
			}
		}

		subQueryset.internals.Where = append(subQueryset.internals.Where, expr.Expr(
			targetField.Name(),
			expr.LOOKUP_IN,
			pks,
		))
	}

	if field, ok := subQueryset.internals.fieldsMap[""]; ok {
		field.Through = targetFieldInfo.Through
	} else {
		subQueryset.internals.Fields = append(
			[]*FieldInfo[attrs.FieldDefinition]{targetFieldInfo},
			subQueryset.internals.Fields...,
		)
	}

	subQueryset.internals.Limit = 0 // preload all objects
	subQueryset.internals.Offset = 0
	var preloadObjects, err = subQueryset.All()
	if err != nil {
		return errors.Wrapf(
			err, "failed to preload %s for %T", preload.Path, r.qs.internals.Model.Object,
		)
	}

	var result = &PreloadResults{
		rowsRaw: preloadObjects,
		rowsMap: make(map[any][]*Row[attrs.Definer], len(preloadObjects)),
	}

	var seenM, ok = r.seen[preload.Path]
	if !ok {
		seenM = &seenObject{
			pks:     make([]any, 0, 1),
			objects: make(map[any]*object, 0),
		}
		r.seen[preload.Path] = seenM
	}

	for _, row := range preloadObjects {
		var (
			rowDefs    = row.Object.FieldDefs()
			parentObj  *object
			parentOk   bool
			primaryVal any
			sourceVal  any
		)
		switch {
		case relThrough == nil:
			var sourceField, ok = rowDefs.Field(preload.Rel.Field().Name())
			if !ok {
				return errors.FieldNotFound.Wrapf(
					"QuerySet.All: preload %q has no field %q in model %T (%#v)",
					preload.FieldName, preload.Field.Name(), r.qs.internals.Model.Object, preload,
				)
			}

			sourceVal, err = sourceField.Value()
			if err != nil {
				return errors.Wrapf(
					err, "failed to get value for source field %q (%#v)", sourceField.Name(), preload,
				)
			}

			primaryVal, err = rowDefs.Primary().Value()
			if err != nil {
				return errors.Wrapf(
					err, "failed to get value for primary field %q (%#v)", sourceField.Name(), preload,
				)
			}

			if slice, ok := result.rowsMap[sourceVal]; ok {
				result.rowsMap[sourceVal] = append(slice, row)
			} else {
				var rows = make([]*Row[attrs.Definer], 0, 1)
				rows = append(rows, row)
				result.rowsMap[sourceVal] = rows
			}

			parentObj, parentOk = seenObj.objects[sourceVal]

		default:
			var defs = row.Through.FieldDefs()
			var sourceField, _ = defs.Field(relThrough.SourceField())
			sourceVal, err = sourceField.Value()
			if err != nil {
				return errors.Wrapf(
					err, "failed to get value for source field %q", sourceField.Name(),
				)
			}

			var targetField, _ = defs.Field(relThrough.TargetField())
			primaryVal, err = targetField.Value()
			if err != nil {
				return errors.Wrapf(
					err, "failed to get value for target field %q", targetField.Name(),
				)
			}

			if slice, ok := result.rowsMap[sourceVal]; ok {
				result.rowsMap[sourceVal] = append(slice, row)
			} else {
				var rows = make([]*Row[attrs.Definer], 0, 1)
				rows = append(rows, row)
				result.rowsMap[sourceVal] = rows
			}

			parentObj, parentOk = seenObj.objects[sourceVal]
		}

		if !parentOk {
			return errors.ValueError.WithCause(fmt.Errorf(
				"QuerySet.All: no parent object found for preload %q with primary key %v (%T) in %v",
				preload.FieldName, sourceVal, sourceVal, seenObj.objects,
			))
		}

		var obj = &object{
			uniqueValue: primaryVal,
			through:     row.Through,
			fieldDefs:   rowDefs,
			obj:         row.Object,
			relations:   make(map[string]*objectRelation),
		}

		seenM.pks = append(seenM.pks, primaryVal)
		seenM.objects[primaryVal] = obj

		relationMap, ok := parentObj.relations[preload.FieldName]
		if !ok {
			relationMap = &objectRelation{
				relTyp:  preload.Rel.Type(),
				objects: orderedmap.NewOrderedMap[any, *object](),
			}
			parentObj.relations[preload.FieldName] = relationMap
		}

		relationMap.objects.Set(primaryVal, obj)
	}

	//preload.Results = result
	//
	//// chain example: "author.books.title" -> "author.books"
	//var chainParts = strings.Join(preload.Chain[:len(preload.Chain)-1], ".")
	//if existing, ok := r.preloads.Get(chainParts); ok {
	//	// if the preload already exists, append the new preload to the existing one
	//	existing = append(existing, preload)
	//	r.preloads.Set(chainParts, existing)
	//} else {
	//	// otherwise, create a new slice with the preload
	//	var p = make([]*Preload, 0, 1)
	//	p = append(p, preload)
	//	r.preloads.Set(chainParts, p)
	//}

	return nil
}

func (r *rows[T]) compile() (count int, rowIter iter.Seq2[*Row[T], error], err error) {

	if r.qs.internals.Preload != nil {
		for _, preload := range r.qs.internals.Preload.Preloads {
			if err := r.queryPreloads(preload); err != nil {
				return 0, nil, fmt.Errorf("failed to query preload %q: %w", preload.FieldName, err)
			}
		}
	}

	var addRelations func(*object, uint64, string) error
	// addRelations is a recursive function that traverses the object and its relations,
	// and sets the related objects on the provided parent object.
	addRelations = func(obj *object, depth uint64, objPath string) error {

		if obj.uniqueValue == nil {
			// panic(fmt.Sprintf("object %T has no primary key, cannot deduplicate relations", obj.obj))
			return fmt.Errorf(
				"object %T has no primary key, cannot deduplicate relations", obj.obj,
			)
		}

		for relName, rel := range obj.relations {
			var relatedObjects = make([]Relation, 0, rel.objects.Len())
			for relHead := rel.objects.Front(); relHead != nil; relHead = relHead.Next() {
				var relatedObj = relHead.Value
				if relatedObj == nil {
					continue
				}

				var newObjPath = objPath
				if newObjPath == "" {
					newObjPath = relName
				} else {
					newObjPath = fmt.Sprintf("%s.%s", newObjPath, relName)
				}

				if err := addRelations(relatedObj, depth+1, newObjPath); err != nil {
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
			if err := setRelatedObjects(relName, rel.relTyp, obj.obj, relatedObjects); err != nil {
				return fmt.Errorf("failed to set related objects for relation %q on object %T: %w", relName, obj.obj, err)
			}
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

	return r.objects.Len(), iter.Seq2[*Row[T], error](func(yield func(*Row[T], error) bool) {

		for head := r.objects.Front(); head != nil; head = head.Next() {
			var obj = head.Value
			if obj == nil {
				continue
			}

			if err := addRelations(obj.object, 0, ""); err != nil {
				yield(nil, fmt.Errorf("failed to add relations for object with primary key %v: %w", obj.object.uniqueValue, err))
				return
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

			var row = &Row[T]{
				Object:          definer.(T),
				ObjectFieldDefs: obj.object.fieldDefs,
				Annotations:     obj.annotations,
				Through:         obj.object.through,
			}

			if !yield(row, nil) {
				return
			}
		}
	}), nil

}

// a chainPart represents a part of a relation chain.
// it contains information about the relation and object.
type chainPart struct {
	// preload     *Preload
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
func (r *rows[T]) buildChainParts(actualField *scannableField) []chainPart {
	// Get the stack of fields from target to parent
	var stack = make([]chainPart, 0)
	for cur := actualField; cur != nil; cur = cur.srcField {
		var (
			inst    = cur.object
			defs    = inst.FieldDefs()
			primary = defs.Primary()
		)

		var (
			pk, err    = primary.Value()
			primaryVal = pk
		)
		if err != nil {
			panic(fmt.Sprintf("error getting primary key value for field %s: %v", cur.chainKey, err))
		}
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

		// var preload *Preload
		if (cur.relType == attrs.RelManyToMany || cur.relType == attrs.RelOneToMany) && fields.IsZero(pk) {

			/*
				the commented out code might be useful in the future
				it is a check to see if the current chain part is being preloaded
				and if it is, we could POSSIBLY, MAYBE properly handle the relation??

				```go
					var preloads = &QuerySetPreloads{}
					if r.qs.internals.Preload != nil {
						preloads = r.qs.internals.Preload
					}

					var hasPreload bool
					preload, hasPreload = preloads.mapping[cur.chainKey]
					if !hasPreload {
						panic(fmt.Sprintf(
							"cannot build chain part for field %s with zero primary key in ManyToMany or OneToMany relation", cur.chainKey,
						))
					}
				```
			*/

			panic(fmt.Sprintf(
				"cannot build chain part for field %s with zero primary key in ManyToMany or OneToMany relation", cur.chainKey,
			))
		}

		stack = append(stack, chainPart{
			// preload:     preload,
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

	// If the root object has a through model, we need to set the unique value
	// based on said through model.
	//
	// this logic is kept in line in [QuerySet.All] before generating the root row.
	if len(stack) > 0 && stack[0].through != nil {
		var uq, err = GetUniqueKey(stack[0].through)
		if err != nil {
			panic(fmt.Sprintf("error getting unique key for through model %T: %v", stack[0].through, err))
		}
		stack[0].uniqueValue = uq
	}

	return stack
}
