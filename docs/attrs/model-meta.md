# Model Meta Information

The `attrs` package provides a way to store meta information about models.

This meta information is retrieved from the actual fields themselves.

The meta information contains the following information:

- The model itself
- The primary field
- The [`FieldDefinitions`](./interfaces.md#fielddefinition) of the model fields
- The forward relations of the model
- The reverse relations of the model
- Any extra information that a model wants to store

## Model Meta- related functions

Functions retrieve, work with or change the model meta information.

### RegisterModel

`RegisterModel(model Definer)`

RegisterModel registers a model to be used for any ORM- type operations.

Models are registered automatically in [django.Initialize](../configuring.md#initializing-the-app) when made available
with the `django.AppConfig.Models()` function, but you can also register them manually if needed.

This function will do a single map lookup to check if the model is already registered.

If the model is already registered, this function will do nothing.

---

### GetModelMeta

`GetModelMeta(model Definer) ModelMeta`

Retrieve the model meta information for the given model.

This function will panic if the model is not registered - you can use `IsModelRegistered` to check if the model is registered.

---

### IsModelRegistered

`IsModelRegistered(model Definer) bool`

Returns true if the model is registered, false otherwise.

---

### GetRelationMeta

`GetRelationMeta(m Definer, name string) (Relation, bool)`

Return meta information about the model's relations.

This function will not panic if the model or relation is not found.

If either of them are not found, this function will return nil and false.

---

### StoreOnMeta

`StoreOnMeta(m Definer, key string, value any)`

Store a value on the model meta.

This function will panic if the model is not registered - you can use `IsModelRegistered` to check if the model is registered.

## Model Meta Interfaces

### ModelMeta

The `ModelMeta` interface is used to retrieve meta information about a model.

It is used to retrieve meta information about the model, such as its relations, and other information that is not part of the model itself.

Models which implement the `Definer` interface can be registered with the `RegisterModel` function.

This will store the model in a global registry, and allow for fast and easy access to the model's meta and field information  
without the model's `FieldDefs` method having to be called.

```go
type ModelMeta interface {
    // Model returns the model for this meta
    Model() Definer

    // Forward returns the forward relations for this model
    // which belong to the field with the given name.
    Forward(relField string) (Relation, bool)

    // Reverse returns the reverse relations for this model
    // which belong to the field with the given name.
    Reverse(relField string) (Relation, bool)

    // ForwardMap returns a copy the forward relations map for this model
    ForwardMap() *orderedmap.OrderedMap[string, Relation]

    // ReverseMap returns a copy of the reverse relations map for this model
    ReverseMap() *orderedmap.OrderedMap[string, Relation]

    // Storage returns a value stored on the model meta.
    //
    // This is used to store values that are not part of the model itself,
    // but are needed for the model or possible third party libraries to function.
    //
    // Values can be stored on the model meta using the `attrs.StoreOnMeta` helper function.
    //
    // A model can also implement the `CanModelInfo` interface to store values on the model meta.
    Storage(key string) (any, bool)

    // Definitions returns the field definitions for the model.
    //
    // This is used to retrieve meta information about fields, such as their type,
    // and other information that is not part of the model itself.
    Definitions() StaticDefinitions
}
```

### CanModelInfo

A model can implement the `CanModelInfo` interface.

This interface is used to store extra information about the model.

This will only be called once, when registering the model.

### Relations

Relations are used to define relations between models.

#### RelationType

There are currently four relation types:

- `RelManyToOne`  -> This is a many to one relationship, also known as a foreign key relationship.
- `RelOneToOne`   -> This is a one to one relationship.
- `RelManyToMany` -> This is a many to many relationship.
- `RelOneToMany`  -> This is a one to many relationship, also known as a reverse foreign key relationship.

#### Relation

The `Relation` interface is used to define relations between models.

This provides a very abstract way of defining relations between models, which can be used to define relations in a more generic way.

It supports defining a through model for a relationship, which can be used to define one to one relations or many to many relations.

One to one relations can also be defined without a through model.

```go
type Relation interface {
    RelationTarget

    Type() RelationType

    // A through model for the relationship.
    //
    // This can be nil, but does not have to be.
    // It can support a one to one relationship with or without a through model,
    // or a many to many relationship with a through model.
    Through() Through
}
```

#### RelationTarget

The `RelationTarget` interface is used to define a relation target.

This contains information about the target model for the relation, which can be used to define the relation in a more generic way.

```go
type RelationTarget interface {
    // From represents the source model for the relationship.
    //
    // If this is nil then the current interface value is the source model.
    From() RelationTarget

    // The target model for the relationship.
    Model() Definer

    // Field retrieves the field in the target model for the relationship.
    //
    // This can be nil, in such cases the relationship should use the primary field of the target model.
    //
    // If a through model is used, the target field should still target the actual target model,
    // the through model should then use this field to link to the target model.
    Field() FieldDefinition
}
```

#### Through

The `Through` interface is used to define a relation between two models.

This provides a very abstract way of defining relations between models, which can be used to define one to one relations or many to many relations.

It acts as an intermediary model which contains a link to the source and target models.

```go
type Through interface {
    // The through model itself.
    Model() Definer

    // The source field for the relation - this is a field in the through model linking to the source model.
    SourceField() string

    // The target field for the relation - this is a field in the through model linking to the target model.
    TargetField() string
}
```
