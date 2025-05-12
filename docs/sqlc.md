# SQLC Plugin

A SQLC plugin has been implemented to help generate [the definer interface](./attrs/interfaces.md#definer) for your models.

It is also capable of:

- Generating extra functions to easily set up the admin panel for the included models
- Generating extra methods to adhere to models.Saver, models.Updater, models.Deleter and models.Reloader interfaces

This plugin is not included in the main package, but can be found in the `cmd` subpackage.

It can be installed by running the following command:

```bash
go install github.com/Nigel2392/go-django/cmd/go-django-definitions@latest
```

This will install the plugin in your `$GOPATH/bin` directory and can later be referenced in your sqlc configuration file.

## Configuration

To use the plugin, you need to add the `go-django-definitions` plugin to your sqlc configuration file.

To learn more about SQLC, please refer to the [SQLC documentation](https://docs.sqlc.dev/).

An example configuration file is provided below:
  
```yaml
version: "2"
plugins:
  - name: go-django-definitions-plugin
    process:
      # go install github.com/Nigel2392/go-django/cmd/go-django-definitions@latest
      cmd: "go-django-definitions"
sql:
  - schema: "./schema.sql"
    queries: "./queries.sql"
    engine: "mysql"
    codegen:
    - out: ./gen
      # The name of the plugin as defined above in the plugins dictionary.
      plugin: go-django-definitions-plugin
      options:
        # required, name of the package to generate
        package: "mypackage" 

        # optional, default is "<package>_definitions.go"
        out: "mydefinitions.go" 

        # optional, default is false, generates extra functions to easily set up the admin panel for the included models
        generate_admin_setup: true

        # optional, default is false, generates extra methods to adhere to models.Saver, models.Updater, models.Deleter and models.Reloader interfaces
        generate_models_methods: true

        # Optional, default is false, generates pointers for fields in the generated struct (only sqlite)
        emit_pointers_for_null: true

        # optional, see https://docs.sqlc.dev/en/stable/reference/config.html
        initialisms: ["id", "api", "url"] 

        # optional, see https://docs.sqlc.dev/en/stable/howto/rename.html
        rename: 

        # overrides, see https://docs.sqlc.dev/en/stable/howto/overrides.html
        overrides:
          - column: "mytable.myfilefield"
            go_type:
              import: "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
              type: "SimpleStoredObject"
              pointer: true
    gen:
      go:
        package: "mypackage"
        out: "./gen"
        emit_json_tags: true
        emit_prepared_queries: true
        emit_result_struct_pointers: true
        emit_interface: true
        query_parameter_limit: 8
        overrides:
          - column: "mytable.myfilefield"
            go_type:
              import: "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
              type: "SimpleStoredObject"
              pointer: true
```

## Directives

The plugin also supports directives; these can be set on a table, or columns of that table.

These directives can allow for extra functionality, such as declaring relations, labels and more.

Multiple directives can be set on a single column or table, and are separated by a semicolon.

For example, to specify a foreign key relation, you can use the `fk` directive.

### Column Directives

#### `fk`

This will then internally link the two tables, and generate the necessary configuration to let Go-Django know about the relation.

If the relation target is in another go package, you can specify the package name as well.

```sql
- `fk:<table>;
```

Or to specify that the relation is in another package:

```sql
- `fk:<table>=github.com/myUser/myPackage.myType`;
```

Pointers in this case are not supported.

#### `label`

This directive allows you to specify a label for the field in the generated struct.

```sql
- `label:<my-label>;`
```

#### `readonly`

This directive allows you to specify that the field is read-only.

```sql
- `readonly:true`
```

### Table Directives

The table directives are similar to the column directives, but are applied to the table as a whole.

The following table directives are supported:

#### `readonly`

This directive allows you to specify that some fields are read-only.

```sql
- `readonly:myField1, myField2`
```

### Example usage of directives

```sql
CREATE TABLE notes (
    id          SERIAL PRIMARY KEY,
    body        TEXT NOT NULL,
    user_id     BIGINT UNSIGNED NOT NULL COMMENT 'fk:users=github.com/Nigel2392/go-django/src/contrib/auth/auth-models.User',
    ticket_id   BIGINT UNSIGNED NOT NULL COMMENT 'fk:tickets',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (ticket_id) REFERENCES tickets(id)
) COMMENT 'readonly:id,created_at,updated_at';

CREATE TABLE tickets (
    id          SERIAL PRIMARY KEY COMMENT 'readonly:true',
    title       VARCHAR(255) NOT NULL COMMENT 'label:"Title" helptext:"The title of the ticket"',
    body        TEXT NOT NULL COMMENT 'label:"Body" helptext:"What is this ticket about?"',

    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'readonly:true',
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'readonly:true'
);
```

In this example, the `notes` table has two foreign key relations, one to the `users` table and another to the `tickets` table.

The `notes` table also has a `readonly` directive applied to the `id`, `created_at` and `updated_at` fields.

The `tickets` table has a `readonly` directive applied to the `id`, `created_at` and `updated_at` fields.

## Result

Executing the SQLC command with the above configuration and schema will generate the appropriate methods to easily work inside of go-django with the models.
