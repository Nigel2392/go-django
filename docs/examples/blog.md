# Pages Example Documentation

Go-Django has provided pre-defined page models for MySQL and SQLite databases.

These can then be extended to create your own custom page model with fields for your specific needs.

The backend for both MySQL and SQLite has to specifically be imported for it to automagically work.

## Introduction

Pages are http-handler- like models, but they have to explicitly be registered to the `pages` application, along with their respective [content type](../../contenttypes.md)

This is best done in your [AppConfig's ready function](../../apps.md#creating-a-new-app).

This allows for the custom admin- area and the rest of the pages application to work with your custom page model.

### Creating a custom page model

As mentioned before, the `*page_models.PageNode` (the model used in the pages app, in a separate package to easily work with multiple databases and prevent import cycles) can be extended to create your own custom page model.

#### Page model interfaces

The custom `Page` model must adhere to the `pages.Page` interface, defined as following (along with other interfaces):

```go
type Page interface {
    // Returns the ID of the page
    ID() int64

    // Return the reference to the underlying generic page node.
    Reference() *models.PageNode
}

type SaveablePage interface {
    Page

    // Allows the page to save itself to the database when performing database operations in the admin area.
    // 
    // Otherwise, only the reference to the page is saved.
    Save(ctx context.Context) error
}

type DeletablePage interface {
    Page

    // Allows the page to delete itself from the database when performing database operations in the admin area.
    // 
    // Otherwise, only the reference to the page is deleted.
    Delete(ctx context.Context) error
}
```

Now that we know what interfaces our custom page model has to adhere to, it is time to create the model itself.

We will extend the [`*page_models.PageNode`](./pages_models.md#pagenode) model, and add a [richtext editor field](../editor/editor.md) to it.

Following that, we will add the required methods to adhere to the `pages.Page` interface.

It is also required for pages to adhere to the [attrs.Definer](../attrs/interfaces.md#definer) interface.

```go
// blog/page.go
package blog

import (
    "github.com/Nigel2392/go-django/src/contrib/editor"
    "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
    "github.com/Nigel2392/go-django/src/core/attrs"

    // Import the required package for working with SQLite3
    _ "github.com/Nigel2392/go-django/src/contrib/pages/backend-sqlite"
)

type BlogPage struct {
    *page_models.PageNode
    Editor *editor.EditorJSBlockData
}

// Adhere to the pages.Page interface
func (b *BlogPage) ID() int64 {
    return b.PageNode.PageID
}

// Adhere to the pages.Page interface
func (b *BlogPage) Reference() *page_models.PageNode {
    return b.PageNode
}

// Adhere to the attrs.Definer interface
func (n *BlogPage) FieldDefs() attrs.Definitions {
    if n.PageNode == nil {
        n.PageNode = &page_models.PageNode{}
    }
    return attrs.Define(n,
        attrs.NewField(n.PageNode, "PageID", &attrs.FieldConfig{
            Primary:  true,
            ReadOnly: true,
        }),
        attrs.NewField(n.PageNode, "Title", &attrs.FieldConfig{
            Label:    "Title",
            HelpText: "How do you want your post to be remembered?",
        }),
        attrs.NewField(n.PageNode, "UrlPath", &attrs.FieldConfig{
            ReadOnly: true,
            Label:    "URL Path",
            HelpText: "The URL path for this blog post.",
        }),
        attrs.NewField(n.PageNode, "Slug", &attrs.FieldConfig{
            Label:    "Slug",
            HelpText: "The slug for this blog post.",
            Blank:    true,
        }),
        attrs.NewField(n, "Editor", &attrs.FieldConfig{
            Default:  &editor.EditorJSBlockData{},
            Label:    "Rich Text Editor",
            HelpText: "This is a rich text editor. You can add images, videos, and other media to your blog post.",
        }),
        attrs.NewField(n.PageNode, "CreatedAt", &attrs.FieldConfig{
            ReadOnly: true,
            Label:    "Created At",
            HelpText: "The date and time this blog post was created.",
        }),
        attrs.NewField(n.PageNode, "UpdatedAt", &attrs.FieldConfig{
            ReadOnly: true,
            Label:    "Updated At",
            HelpText: "The date and time this blog post was last updated.",
        }),
    )
}
```

To display your pages to users the page, much like a `http.Handler` must provide a `ServeHTTP` method.

This method is called when the page is being served to the user.

```go
// blog/page.go
func (b *BlogPage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "<html><head><title>Blog Page</title></head><body>")
    // Serve the blog page here.
    fmt.Fprintf(w, "<h1>%s</h1>\n", b.Title)
    var rendered, err = b.Editor.Render()
    if err != nil {
        logger.Errorf("Error rendering blog page: %v\n", err)
        return
    }
    fmt.Fprintf(w, "%s\n", rendered)
    fmt.Fprintln(w, "</body></html>")
}
```

### Defining the database queries

As mentioned before, we must also provide `Save` and `Delete` methods to our custom page model.

These methods are used to save and delete the page from the database, respectively, otherwise only the reference to the page is saved or deleted.

```go
// blog/page.go
func (b *BlogPage) Save(ctx context.Context) error {
    var err error
    if b.ID() == 0 {
        var id int64
        id, err = createBlogPage(b.Title, b.Editor)
        b.PageID = id
    } else {
        err = updateBlogPage(b.PageNode.PageID, b.Title, b.Editor)
    }
    if err != nil {
        logger.Errorf("Error saving blog page: %v\n", err)
    }
    return err
}
```

We will create a separate file for the SQL queries themselves.

Currently, we will settle for SQLite3 but this is not a requirement, if you wish to use MySQL all queries must be implemented with the required syntax for MySQL.

The attributes of the `page_models.PageNode` should not be added to the SQL queries, as they are handled by the pages app itself, in a separate table.

#### SQLite3 Queries

```go
// blog/sqlite.go
package blog

import (
    "database/sql"

    "github.com/Nigel2392/go-django/src/contrib/editor"
    "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
    "github.com/pkg/errors"
)

const (
    createTable = `CREATE TABLE IF NOT EXISTS blog_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT,
    editor TEXT
    )`
    insertPage = `INSERT INTO blog_pages (title, editor) VALUES (?, ?)`
    updatePage = `UPDATE blog_pages SET title = ?, editor = ? WHERE id = ?`
    selectPage = `SELECT id, title, editor FROM blog_pages WHERE id = ?`
)

func CreateTable(db *sql.DB) error {
    _, err := db.Exec(createTable)
    return err
}

func createBlogPage(title string, richText *editor.EditorJSBlockData) (id int64, err error) {
    res, err := blog.DB.Exec(insertPage, title, richText)
    if err != nil {
        return 0, err
    }

    id, err = res.LastInsertId()
    if err != nil {
        return 0, err
    }

    return id, nil
}

func updateBlogPage(id int64, title string, richText *editor.EditorJSBlockData) error {
    var _, err = blog.DB.Exec(updatePage, title, richText, id)
    return err
}

func getBlogPage(parentNode page_models.PageNode, id int64) (*BlogPage, error) {
    var page = &BlogPage{
        PageNode: &parentNode,
    }
    if blog.DB == nil {
        return nil, errors.New("blog.DB is nil")
    }
    var row = blog.DB.QueryRow(selectPage, id)
    var err = row.Err()
    if err != nil {
        return nil, errors.Wrapf(
            err, "Error getting blog page with id %d (%T)", id, id,
        )
    }

    err = row.Scan(&page.PageNode.PageID, &page.Title, &page.Editor)
    if err != nil {
        return nil, errors.Wrapf(
            err, "Error scanning blog page with id %d (%T)", id, id,
        )
    }

    return page, err
}
```

### The `NewBlogPageConfig` function

Most of the work for the blog app itself is now done.

We only need to define the `django.AppConfig` object to properly be able to register it to Go-Django.

First we will handle all imports.

```go
// blog/app.go
package blog

import (
    "context"
    "database/sql"
    "net/http"

    django "github.com/Nigel2392/go-django/src"
    "github.com/Nigel2392/go-django/src/apps"
    "github.com/Nigel2392/go-django/src/contrib/admin"
    "github.com/Nigel2392/go-django/src/contrib/pages"
    "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
    "github.com/Nigel2392/go-django/src/core/contenttypes"
    "github.com/Nigel2392/go-django/src/core/trans"
)

```

We can now create a stub for the blog AppConfig.

<pre>
var blog *apps.DBRequiredAppConfig

func NewAppConfig() *apps.DBRequiredAppConfig {
    var appconfig = apps.NewDBAppConfig("blog")
    appconfig.Init = func(settings django.Settings, db *sql.DB) error {
        return CreateTable(db)
    }
    appconfig.Ready = func() error {
        // ...<a href="#registering-the-blog-page-model">Registering the blog page model</a>
        blog = appconfig
        return nil
    }
    return appconfig
}
</pre>

### Registering the blog page model

The blog page model must be registered to the pages app.

This is done by calling the `pages.Register` function with a [`*pages.PageDefinition`](./contenttypes.md#page-definitions-and-registration) object.

This is an extension of the [`contenttypes.ContentTypeDefinition`](../../contenttypes.md#content-type-definition) object.

It also provides the panels which are shown on page creation, or when updating a page, as well as setting specific parent page types by providing their content type aliases.

```go
// The underlying content type definition for the blog page model
var definition = &contenttypes.ContentTypeDefinition{
    GetLabel:       trans.S("Blog Page"),
    GetDescription: trans.S("A blog page with a rich text editor."),
    ContentObject:  &BlogPage{},
}

// Panels for the blog page model when creating a new page
// 
// This contains fields from the blog page model, as well as the underlying page node model.
var addPanels = func(r *http.Request, page pages.Page) []admin.Panel {
    return []admin.Panel{
        admin.TitlePanel(
            admin.FieldPanel("Title"),
        ),
        admin.MultiPanel(
            admin.FieldPanel("UrlPath"),
            admin.FieldPanel("Slug"),
        ),
        admin.FieldPanel("Editor"),
    }
}

// Panels for the blog page model when editing a page
//
// This contains fields from the blog page model, as well as the underlying page node model.
var editPanels = func(r *http.Request, page pages.Page) []admin.Panel {
    return []admin.Panel{
        admin.TitlePanel(
            admin.FieldPanel("Title"),
        ),
        admin.MultiPanel(
            admin.FieldPanel("UrlPath"),
            admin.FieldPanel("Slug"),
        ),
        admin.FieldPanel("Editor"),
        admin.FieldPanel("CreatedAt"),
        admin.FieldPanel("UpdatedAt"),
    }
}

// The allowed parent page types for the blog page model
var allowedParentPageTypes = []string{
    // "github.com/yourname/yourproject/custom.Page",
}

// The allowed child page types for the blog page model
var allowedChildPageTypes = []string{
    // "github.com/yourname/yourproject/custom.Page",
}

// Disallow creation of this model
var disallowCreate = false

// Disallow this page type to be a root page
var disallowRoot = false

// Serve the page with this view instead
var servePage func(p pages.Page) pages.PageView = nil

pages.Register(&pages.PageDefinition{
    ContentTypeDefinition: definition,
    AddPanels: addPanels,
    EditPanels: editPanels,
    ParentPageTypes: allowedParentPageTypes,
    ChildPageTypes: allowedChildPageTypes,
    DisallowCreate: disallowCreate,
    DisallowRoot: disallowRoot,
    ServePage: servePage,
    GetForID: func(ctx context.Context, ref page_models.PageNode, id int64) (pages.Page, error) {
        return getBlogPage(ref, id)
    },
})
```

### Creating your Go-Django web application

Now that we have our blog app set up, we can create a new Go-Django web application.

The page app does not know from which URL it should be served, [this has to be explicitly configured](./readme.md#routing-and-url-handling).

First we will handle all imports.

```go
// main.go
package main

import (
    "database/sql"

    "github.com/yourname/yourproject/blog"
    django "github.com/Nigel2392/go-django/src"
    "github.com/Nigel2392/go-django/src/contrib/admin"
    "github.com/Nigel2392/go-django/src/contrib/auth"
    "github.com/Nigel2392/go-django/src/contrib/editor"
    "github.com/Nigel2392/go-django/src/contrib/pages"

    "github.com/Nigel2392/go-django/src/contrib/session"

    _ "github.com/Nigel2392/go-django/src/contrib/pages/backend-sqlite"
    _ "github.com/mattn/go-sqlite3"
)
```

<pre>
func main() {
    // ...<a href="#defining-the-go-django-webapplication">Defining the Go-Django webapplication</a>
    // ...<a href="#setup-the-page-apps-route">Setup the page app's route</a>
    // ...<a href="#initialize-the-go-django-webapplication">Initialize the Go-Django webapplication</a>
    // ...<a href="#serve-the-go-django-webapplication">Serve the Go-Django webapplication</a>
}
</pre>

#### Define the Go-Django webapplication

The Go-Django app is easily initialized.

We will have to add a few other apps to the app list, such as `sessions`, `auth`, `admin`, `pages`, and `blog`, as well as initialize the database connection.

```go
var app = django.App(
    django.Configure(map[string]interface{}{
        django.APPVAR_ALLOWED_HOSTS: []string{"*"},
        django.APPVAR_DEBUG:         false,
        django.APPVAR_HOST:          "127.0.0.1",
        django.APPVAR_PORT:          "8080",
        django.APPVAR_DATABASE: func() *sql.DB {
            var db, err = drivers.Open("sqlite3", "./.private/db.sqlite3")
            if err != nil {
                panic(err)
            }
            return db
        }(),
    }),
    django.Apps(
        session.NewAppConfig,
        auth.NewAppConfig,
        admin.NewAppConfig,
        pages.NewAppConfig,
        editor.NewAppConfig,
        blog.NewAppConfig,
    ),
)
```

#### Setup the page app's route

The page app has to be served from a specific URL.

Calling the `pages.SetRoutePrefix` will set the URL prefix for the page app, as well as providing any required functionality to link back to live pages.

```go
pages.SetRoutePrefix("/blog")
```

#### Initialize the Go-Django webapplication

The Go-Django webapplication is initialized by calling the `app.Initialize` method.

```go
var err = app.Initialize()
if err != nil {
    panic(err)
}
```

#### Serve the Go-Django webapplication

The Go-Django webapplication is served by calling the `app.Serve` method.

```go
err = app.Serve()
if err != nil {
    panic(err)
}
```

### Create a superuser for the admin area

To access the admin area, a superuser has to be created.

This can be done by executing main.go with the `createuser` argument, and the `-s` flag set.

```sh
go run main.go createuser -s
```

This will prompt you to enter a username, email, and password for the superuser.

### Running the application

After running the application, you should be able to create new blog pages from the Go-Django admin area.

These pages can then be accessed from `http://127.0.0.1:8080/blog`.

You can serve the app by executing the following command:

```sh
go run main.go
```

#### The resulting project structure

The resulting project structure should look like this:

```sh
github.com/yourname/yourproject
|-- .private
|   `-- db.sqlite3
|-- blog
|   |-- app.go
|   |-- page.go
|   `-- sqlite.go
|-- main.go
`-- go.mod
```
