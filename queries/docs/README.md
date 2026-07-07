# Queries Documentation

Welcome to the `queries` module documentation. This module provides robust database querying, model definition capabilities, and relationship management for Go-Django.

## Getting Started

- [Getting Started](./getting_started.md) - A quick introduction to connecting and using the ORM.
- [Interfaces](./interfaces.md) - Overview of the core interfaces that power the ORM.
- [Drivers](./drivers.md) - Supported database drivers and how to register them.

## Models and Fields

- [Defining Models](./models/models.md) - How to structure your structs and implement `FieldDefs`.
- [The `models.Model` Base](./models/model.md) - Using the embedded base model and its lifecycle hooks.
- [Proxy Models](./models/proxy_models.md) - How to inherit and extend models without creating new tables.
- [Fields](./fields.md) - Using the `fields` package to declare column types and constraints.
- [Virtual Fields](./virtual_fields.md) - Adding non-database, computed properties to your models.
- [Relations](./relations/relations.md) - Working with Foreign Keys, One-to-One, Many-to-Many, and Reverse relations.

## Querying the Database

- [Basic Querying](./querying.md) - An overview of common ORM methods (`CreateObject`, `DeleteObject`, etc.).
- [The QuerySet API](./queryset/queryset.md) - Chaining methods like `Filter`, `Exclude`, `Select`, and `OrderBy`.
- [Writing Complex Queries](./queryset/writing_queries.md) - Using Q objects and nested conditions.

## Expressions and Lookups

- [Expressions](./expressions/expressions.md) - Using `F()` expressions and aggregate functions.
- [Cases](./expressions/cases.md) - Implementing `CASE WHEN` logic in your queries.
- [Lookups](./expressions/lookups.md) - Field lookups (e.g., `__gt`, `__in`, `__icontains`) for filtering data.
