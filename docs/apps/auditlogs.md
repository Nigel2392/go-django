# Auditlogs app

Auditlogs are a way to keep track of certain actions which have been performed by the user.
This is implemented by logging these actions to a separate database table.
The Auditlogs app provides a user interface to view these logs.

Some apps automatically log actions, for example - each action performed in the admin interface is logged.

Auditlogs can be used anywhere and for various purposes, for example:

- Tracking changes made to important data
- Monitoring user activity for security purposes
- Keeping a history of actions for auditing and compliance

## Viewing Auditlogs

To view the auditlogs, navigate to the Auditlogs app in the main menu of the admin interface.

This URL would look something like this: `https://yourdomain.com/admin/auditlogs/`

Here, you will see a list of all the auditlogs, along with details such as the user who performed the action, the action performed, the timestamp, and any additional information. You can filter and search through the logs to find specific entries.

Auditlogs also allow custom links in the list of auditlogs to quickly navigate to or interact with the related object.

## Customizing Auditlog Definitions

Auditlog definitions can be customized to log specific actions or data as needed.

These basically prepare the data to be shown in the admin interface, as well as what data to log in the first place.

These definitions also define what custom links are shown in the auditlog list for each log entry.

Custom definitions can be created by adhering to the `auditlogs.Definition` interface.

An example of this interface and related interfaces can be seen below:

```golang
type LogEntryAction interface {
    Icon() string
    Label() string
    URL() string
}

type Definition interface {
    TypeLabel(r *http.Request, typeName string) string
    GetLabel(r *http.Request, logEntry LogEntry) string
    FormatMessage(r *http.Request, logEntry LogEntry) any // string | template.HTML
    GetActions(r *http.Request, logEntry LogEntry) []LogEntryAction
}
```

These definitions can then be registered in the `init()` function of your app, like so:

```golang
func init() {
    auditlogs.RegisterDefinition("example.delete_model", &YourCustomAuditlogDefinition{})
}
```

This will ensure that your custom auditlog definitions are used when logging and displaying auditlogs for your app.

These custom definitions can then be used when creating auditlog entries in your app.

## Logging to the Auditlogs app

For example, to log a message of type `example.delete_model`, you would do something like this:

```golang
import (
    "context"
    "github.com/Nigel2392/mux/middleware/authentication"
    "github.com/Nigel2392/go-django/src/core/logger"
    auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
)

var contextWithUser = authentication.ContextWithUser(
    context.Background(),
    myUserInstance,
)

logEntryId, err := auditlogs.Log(
    contextWithUser,         // context for the operation, can include `User` object
    "example.delete_model",  // type of the log entry
    logger.INF,              // level of the log entry
    myModelInstance,         // the related object (can be nil)
    map[string]interface{}{  // additional data to log
        "title":    myModelInstance.Title,
    },
)
```

An important fact to note about the above code is that the user performing the action is taken from the context.

This makes it easy to log actions using just the request's context (`request.Context()`) in web handlers, but
also means it requires an extra step to set the user in the context when logging outside of web handlers.

The auditlogs also uses the `queries` package to log the actions to the database, meaning any transactions present in the context will be honored.

