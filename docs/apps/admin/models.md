# Admin Models, Forms, Panels, and Columns

Models registered with an admin app get full CRUD views (list, add, edit, single delete, and bulk delete), customizable form layouts via panels, inline relation formsets, sortable/actionable list columns, filters, search, and export.

For site-level and app-level configuration, see:

- [Admin Application (`admin`)](./app.md)
- [Registering Apps in Admin](./apps.md)

---

## Registering Models (`ModelOptions` and `ModelDefinition`)

Register models inside `admin.RegisterApp(...)` or via `appDef.Register(ModelOptions{...})`. Every registered model must satisfy two prerequisites:

1. It must be a struct (or pointer to struct) with at least one field.
2. It must have a registered `contenttypes.ContentTypeDefinition` (automatically created for every model in your `AppConfig.ModelObjects`, or registered manually via `contenttypes.Register`).

```go
type ModelOptions struct {
	Name                string
	MenuIcon            func(ctx context.Context) string
	MenuOrder           int
	MenuLabel           func(ctx context.Context) string
	DisallowList        bool
	DisallowCreate      bool
	DisallowEdit        bool
	DisallowDelete      bool
	AddView             FormViewOptions
	EditView            FormViewOptions
	ListView            ListViewOptions
	DeleteView          DeleteViewOptions
	RegisterToAdminMenu any
	Labels              map[string]func(ctx context.Context) string
	Model               attrs.Definer
}
```

- **`Name`**: URL-safe identifier used in `/admin/apps/<app_name>/<model_name>/`. If empty, defaults to `attrs.ColumnName(cType.Name())`. Must match `^[a-zA-Z0-9_]+$` (checked via `assert.True` during `Register`).
- **`RegisterToAdminMenu`**: `true` adds a model link under `"admin:register_menu_item:<app_name>"`, or pass a custom hook string. `false`/`nil` hides it from the sidebar.
- **`DisallowList` / `DisallowCreate` / `DisallowEdit` / `DisallowDelete`**: Disables the corresponding admin action, hides its UI links/buttons, and fails direct HTTP requests with `403 Forbidden` (`autherrors.Fail`).
- **`Labels`**: Model-wide field label overrides; view-specific `ViewOptions.Labels` take precedence.

### `ModelDefinition` and Helpers

`Register` returns a `*ModelDefinition` wrapping `ModelOptions`:

```go
func (o *ModelDefinition) App() *AppDefinition
func (o *ModelDefinition) NewInstance() attrs.Definer
func (o *ModelDefinition) GetName() string
func (o *ModelDefinition) Label(ctx context.Context) string
func (o *ModelDefinition) PluralLabel(ctx context.Context) string
func (o *ModelDefinition) TypeName() string
func (o *ModelDefinition) GetColumn(ctx context.Context, opts ListViewOptions, field string) list.ListColumn[attrs.Definer]
func (o *ModelDefinition) GetLabel(opts ViewOptions, field string, default_ string) func(ctx context.Context) string
func (o *ModelDefinition) FormatColumn(field string) any
func (m *ModelDefinition) ModelFields(opts ViewOptions, instance attrs.Definer) []attrs.Field
func (m *ModelDefinition) GetInstance(ctx context.Context, identifier any) (attrs.Definer, error)
```

Two package-level helpers locate definitions and build model URLs:

- **`admin.FindDefinition(model any, appName ...string) *ModelDefinition`**: Accepts an `attrs.Definer`, a model name `string`, or a `contenttypes.GenericContentType` (plus optional `appName`).
- **`admin.ModelURL(model attrs.Definer, extra []string, vars ...any) string`**: Reverses `"admin:apps:model"` with optional sub-route segments (`extra`), e.g. `admin.ModelURL(&Product{}, []string{"add"})` or `admin.ModelURL(&Product{}, []string{"edit"}, product.ID)`.

---

## View Options (`AddView`, `EditView`, `ListView`, `DeleteView`)

### `ViewOptions`

```go
type ViewOptions struct {
	Fields  []string
	Exclude []string
	Labels  map[string]func(ctx context.Context) string
}
```

**Important**: `Fields` and `Exclude` are mutually exclusive. Setting both triggers a panic (`assert.Fail("Fields and Exclude cannot be used together")`).

### `FormViewOptions`

Used by `AddView` and `EditView`:

```go
type FormViewOptions struct {
	ViewOptions
	Widgets      map[string]widgets.Widget
	GetForm      func(req *http.Request, instance attrs.Definer, fields []string) modelforms.ModelForm[attrs.Definer]
	FormInit     func(instance attrs.Definer, form modelforms.ModelForm[attrs.Definer])
	GetHandler   func(adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) views.View
	SaveInstance func(context.Context, attrs.Definer) error
	Panels       []Panel
}
```

- **`Panels`**: When empty, `FormViewOptions.SetupDefaults` looks for `GetAddPanels() []Panel` (on `AddView`) or `GetEditPanels() []Panel` (on `EditView`) on your model struct.
- **`GetForm`**: Overrides default form construction (`GetAdminForm`).
- **`FormInit`**: Runs right after form creation to add extra fields (like `PasswordConfirm`), set custom validators, or adjust field ordering/visibility.
- **`Widgets`**: Map of field names to `widgets.Widget` overrides applied via `form.AddWidget`.
- **`SaveInstance`**: Overrides default `models.SaveModel(ctx, instance)`.
- **`GetHandler`**: Replaces the built-in `FormView` completely with a custom `views.View`.

### `ListViewOptions`

```go
type ListViewOptions struct {
	ViewOptions
	PerPage     uint64
	Ordering    []string
	BulkActions []BulkAction
	Columns     map[string]list.ListColumn[attrs.Definer]
	Format      map[string]func(v any) any
	Filters     []filters.FilterSpec[attrs.Definer]
	Search      *searches.SearchOptions
	GetList     func(r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, results []attrs.Definer) (list.StringRenderer, error)
	GetHandler  func(adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) views.View
	GetQuerySet func(adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) *queries.QuerySet[attrs.Definer]
	Prefetch    Prefetch
}
```

- **`PerPage`**: Defaults to `25`.
- **`Fields` / `Exclude`**: When `Fields` is unset during `OnRegister`, defaults to all non-M2M / non-through-O2O fields on the model.
- **`Columns` & `Format`**: `Columns` provides full `list.ListColumn[attrs.Definer]` overrides per field name (including virtual columns not backed directly by a model field, such as `"Email__Edit"` or `"Actions"`). `Format` provides simpler `func(v any) any` value formatters.
- **`Filters`**: Adds a [Filters](../filters.md) slide-out side panel (`SidePanelFilters`).
- **`Search`**: Registers the model with [Global Search](./app.md#global-search).
- **`Prefetch`**: Applies `SelectRelated` and `Preload` to list queries (and bulk delete queries).
- **`BulkActions`**: Enables row-selection checkboxes and bulk action buttons when the user has `"admin:bulk_actions"`.

### `DeleteViewOptions`

```go
type DeleteViewOptions struct {
	GetHandler      func(adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instances []attrs.Definer) views.View
	DeleteInstances func(context.Context, []attrs.Definer) error
}
```

Both single deletion (`admin:apps:model:delete`) and bulk deletion (`admin:apps:model:bulk_delete?pk_list=1&pk_list=2`) use `AdminDeleteView` (rendering `"admin/views/models/delete.tmpl"`). On `POST`, `AdminDeleteView`:

1. Verifies `"admin:delete"` object permission and `!model.DisallowDelete`.
2. Fires `AdminModelHookDelete` hooks.
3. Calls `DeleteInstances` if set, or deletes via `queries.GetQuerySetWithContext(...).Delete(instances...)`.
4. Flashes a status message and redirects to `next` or the model's list view.

---

## Complete Model Registration Example

```go
admin.RegisterApp(
	"blog",
	admin.AppOptions{
		RegisterToAdminMenu: true,
		MenuLabel:           trans.S("Blog"),
		MenuOrder:           10,
	},
	admin.ModelOptions{
		Name:                "article",
		Model:               &Article{},
		RegisterToAdminMenu: true,
		AddView: admin.FormViewOptions{
			ViewOptions: admin.ViewOptions{
				Exclude: []string{"ID", "CreatedAt", "UpdatedAt"},
			},
			Panels: []admin.Panel{
				admin.TitlePanel(
					admin.FieldPanel("Title"),
				).WithOutputFields("Slug"),
				admin.RowPanel(
					admin.FieldPanel("Slug"),
					admin.FieldPanel("PublishedAt"),
				),
				admin.FieldPanel("Body"),
			},
		},
		EditView: admin.FormViewOptions{
			ViewOptions: admin.ViewOptions{
				Exclude: []string{"ID"},
			},
		},
		ListView: admin.ListViewOptions{
			PerPage:  25,
			Ordering: []string{"-CreatedAt"},
			BulkActions: []admin.BulkAction{
				admin.BulkActionDelete,
			},
			ViewOptions: admin.ViewOptions{
				Fields: []string{"Title", "Author", "CreatedAt", "Actions"},
			},
			Prefetch: admin.Prefetch{
				SelectRelated: []string{"Author"},
			},
			Columns: map[string]list.ListColumn[attrs.Definer]{
				"CreatedAt": columns.TimeSinceColumn[attrs.Definer](
					trans.S("Created"), "CreatedAt",
				),
				"Actions": &columns.ListActionsColumn[attrs.Definer]{
					Heading: trans.S("Actions"),
					Actions: []*columns.ListAction[attrs.Definer]{
						{
							Text: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
								return trans.T(r.Context(), "Edit")
							},
							URL: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
								return django.Reverse("admin:apps:model:edit", "blog", "article", defs.Get("ID"))
							},
						},
					},
				},
			},
			Search: &searches.SearchOptions{
				Fields: []searches.SearchField{
					{Name: "Title", Lookup: expr.LOOKUP_ICONTANS},
				},
			},
		},
	},
)
```

---

## `AdminForm` and `FormDefiner`

```go
type FormDefiner[T attrs.Definer] interface {
	attrs.Definer
	AdminForm(r *http.Request, app *AppDefinition, model *ModelDefinition) modelforms.ModelForm[T]
}

type AdminForm[T1 modelforms.ModelForm[T2], T2 attrs.Definer] struct {
	Form    T1
	Panels  []Panel
	Request *http.Request
	// ...
}

func NewAdminForm[T1 modelforms.ModelForm[T2], T2 attrs.Definer](r *http.Request, form T1, panels ...Panel) *AdminForm[T1, T2]
```

If a model implements `FormDefiner[T]`, `GetAdminForm` delegates form creation to `instance.AdminForm(r, app, model)`.

Key behaviors of `AdminForm`:

- **Field filtering from panels**: When `Panels` is non-empty, `AdminForm.Load()` collects every parent field referenced by non-`FormPanel` panels and removes unreferenced fields from the parent `ModelForm` via `DeleteField`. (If `Panels` is empty, `Load()` creates one `FieldPanel` per field.)
- **Transactional saving**: `AdminForm.Save()` / `SaveFunc()` starts a database transaction (`queries.StartTransaction`) when `APPVAR_DATABASE` is configured, commits only if all parent and inline forms save with zero errors, and rolls back on any error.
- **Pre/post cluster save ordering (`ClusterOrderableForm`)**:

```go
type FORM_ORDERING int

const (
	FORM_ORDERING_PRE  FORM_ORDERING = -1
	FORM_ORDERING_NONE FORM_ORDERING = 0
	FORM_ORDERING_POST FORM_ORDERING = 1
)

type ClusterOrderableForm interface {
	FormOrder() FORM_ORDERING
}
```

Nested forms with `FORM_ORDERING_PRE` (such as `ManyToOne` or direct `OneToOne` `ModelFormPanel`s) save *before* the parent model, while `FORM_ORDERING_POST` forms (such as `OneToMany`, `ManyToMany`, or reverse/through `OneToOne` panels) receive the parent's primary key automatically and save *after* the parent.

---

## Form Panels

Panels control both visual layout (`PanelBoundForm`) and nested formsets (`FormPanel`) or custom cross-field validation (`ValidatorPanel`):

```go
type Panel interface {
	Fields() []string
	ClassName() string
	Class(classes string) Panel
	Comparison(ctx context.Context, oldInstance attrs.Definer, newInstance attrs.Definer) (compare.Comparison, error)
	Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField, formset FormSetObject) BoundPanel
}
```

### `FieldPanel` and `HiddenFieldPanel`

```go
func FieldPanel(fieldname string, className ...string) Panel
func HiddenFieldPanel(fieldname string, className ...string) Panel
```

`FieldPanel` renders a collapsible `WidgetPanelComponent` containing the bound field's label, errors, help text, and widget.

**Caveat on `HiddenFieldPanel`**: In `contrib/admin/form_panels.go`, `HiddenFieldPanel` currently constructs and returns `&fieldPanel{...}` identically to `FieldPanel`. Because `WidgetPanelComponent` omits `.panel__heading` and `.panel__help` only when `BoundField.Hidden()` is `true`, you must also hide the form field itself (for example with `fields.Hide(true)` in `attrs.Config.FormField` or `field.Hide(true)` inside `FormInit`) to render it purely as a hidden input.

### `TitlePanel`

```go
func TitlePanel(panel Panel, classname ...string) OutputtableTitlePanel
```

Wraps a primary title `FieldPanel` in `.panel-title` with Stimulus `data-controller="titlepanel"`. Chain `.WithOutputFields("Slug")` to wire automatic slug generation into target field IDs on the client.

### `RowPanel`, `LabeledRowPanel`, `PanelGroup`, and `LabeledPanelGroup`

```go
func RowPanel(panels ...Panel) Panel
func LabeledRowPanel(label any, helpText any, panels ...Panel) Panel
func PanelGroup(panels ...Panel) Panel
func LabeledPanelGroup(label any, helpText any, panels ...Panel) Panel
```

- `RowPanel` / `LabeledRowPanel`: Arrange child panels horizontally in a row (`.panel-row`), optionally with a section heading and help text.
- `PanelGroup` / `LabeledPanelGroup`: Group child panels vertically (`.panel-group`), optionally with a heading and help text.

Both implement `MultiPanel`, `FormPanel`, and `ValidatorPanel`, so child `ModelFormPanel`s and validator panels work seamlessly inside rows and groups.

### `AlertPanel`

```go
type AlertType string

const (
	ALERT_INFO    AlertType = "info"
	ALERT_SUCCESS AlertType = "success"
	ALERT_WARNING AlertType = "warning"
	ALERT_ERROR   AlertType = "danger"
)

type AlertPanel struct {
	Type         AlertType
	Label        any
	HTML         any
	TemplateFile string
	Classnames   string
}
```

Renders an informational callout inside a form using either `HTML` (a `string`, `template.HTML`, or `trans.TranslateFunc`) or a custom `TemplateFile` rendered with `*admin.PanelContext`. Defaults to `ALERT_INFO`.

### `TabbedPanel` and `PanelTab`

```go
func TabbedPanel(tabs ...TabPanel) Panel
func PanelTab(title any, panels ...Panel) TabPanel
```

Renders a Stimulus-driven tabbed container (`data-controller="tabpanel"`). Tabs can contain any panels, including nested `ModelFormPanel`s, rows, groups, or nested `TabbedPanel`s. Tabs whose bound child panels all evaluate to empty are omitted.

### `ModelFormPanel` and `ModelPanel` (Inline Formsets)

```go
type ModelFormPanel[TARGET attrs.Definer, FORM modelforms.ModelForm[TARGET]] struct {
	Form           func() FORM
	SaveInstance   func(ctx context.Context, form FORM, instance TARGET) error
	TargetType     TARGET
	MaxNum         int
	MinNum         int
	DisallowAdd    bool
	DisallowRemove bool
	Extra          int
	Classname      string
	SubClassname   string
	FieldName      string
	Panels         []Panel
}

func ModelPanel[TARGET attrs.Definer, FORM modelforms.ModelForm[TARGET]](
	fieldName string,
	targetType TARGET,
	options ...func(*ModelFormPanel[TARGET, FORM]),
) *ModelFormPanel[TARGET, FORM]
```

`ModelFormPanel` manages inline editing of related models via `formsets.BaseFormSet`:

- Supports `RelOneToMany` (child models with FK back to parent), `RelManyToOne`, `RelManyToMany` (including `Through` models with duplicate-target validation), and `RelOneToOne` (direct, reverse, and through).
- `MaxNum` defaults to `100` when `0`; for `ManyToOne` and `OneToOne`, `MinNum` and `MaxNum` are clamped to `<= 1` (`Extra` to `0`).
- `SubClassname` sets extra classes (for example `"collapsed"`) on each child inline form card; `Classname` styles the outer panel (for example `"collapsible"`).
- Automatically injects deletion (`__DELETED__`), ordering, and reverse FK fields if omitted from `Panels`.

### `JSONDetailPanel`

```go
type JSONDetailPanel struct {
	FieldName string
	Classname string
	Ordering  func(r *http.Request, fields map[string]forms.BoundField) []string
	Labels    func(r *http.Request, fields map[string]forms.BoundField) map[string]any
	Widgets   func(r *http.Request, fields map[string]forms.BoundField) map[string]widgets.Widget
}
```

A read-only panel that unmarshals a `string` or `[]byte` JSON object field and renders each key as a read-only sub-field (`<FieldName>__<key>`), ordered by `Ordering` (then alphabetically), with optional custom labels and widgets.

### `PanelTree` and Panel Utilities

```go
func NewPanelTree(panels ...Panel) *PanelTree
func (p *PanelTree) FindNode(contentPath []string) *PanelTree
func PanelClass(className string, panel Panel) Panel
func PanelComparison(ctx context.Context, panels []Panel, oldInstance, newInstance attrs.Definer, mustWrap ...bool) (compare.Comparison, error)
```

`NewPanelTree` builds a navigable tree keyed by `<lowercase_type>-<index>` (for example `"rowpanel-1"`, `"fieldpanel-0"`), used by comment/annotation paths and programmatic panel inspection.

---

## List Columns (`contrib/admin/components/columns`)

The `github.com/Nigel2392/go-django/contrib/admin/components/columns` package provides admin list columns and sorting:

### `ListActionsColumn`

Renders a per-row dropdown menu (`data-controller="dropdown"`) of links or buttons:

```go
type ListAction[T attrs.Definer] struct {
	Classname string
	Show      func(r *http.Request, defs attrs.Definitions, row T) bool
	Text      func(r *http.Request, defs attrs.Definitions, row T) string
	URL       func(r *http.Request, defs attrs.Definitions, row T) string
	Attrs     func(r *http.Request, defs attrs.Definitions, row T) templ.Attributes
}

type ListActionsColumn[T attrs.Definer] struct {
	Heading func(c context.Context) string
	Actions []*ListAction[T]
}
```

### `TimeSinceColumn`

```go
func TimeSinceColumn[T attrs.Definer](label any, field string, hoverFormat ...string) list.ListColumn[T]
func FormatTimeSince(ctx context.Context, t time.Time, diffFrom time.Time) string
func FormatTimeDifference(ctx context.Context, t time.Time, diffFrom time.Time) ([]string, TimeInformation)
```

Formats `time.Time` fields as relative timestamps (e.g. `"2 hours and 15 minutes ago"`, or `"Never"` warning badge when zero) with a Stimulus tooltip showing the formatted datetime (`trans.LONG_TIME_FORMAT` by default).

### `SortableColumnBuilder`

`ModelListHandler` wraps list columns with `columns.NewSortableColumnBuilder(model.Model)`, rendering clickable `?sort=<field>` / `?sort=-<field>` headers and applying `qs.OrderBy(...)`:

```go
func NewSortableColumnBuilder[T attrs.Definer](
	model T,
	isSortField ...func(f attrs.FieldDefinition) bool,
) *SortableColumnBuilder[T]
```

**Minor caveat**: `NewSortableColumnBuilder` currently checks `if len(isSortField) > 1` rather than `> 0` before reading `isSortField[0]`. When called with 0 or 1 argument, it uses the default predicate (`f.Rel() == nil`), making all non-relation model fields sortable.

### Automatic `TitleFieldColumn` and `RowSelectColumn`

`ModelListHandler` automatically wraps the list's title column with:

1. `list.TitleFieldColumn(...)`: Links the first column directly to `admin:apps:model:edit` whenever `!model.DisallowEdit` and the user has `"admin:edit"` object permission.
2. `list.RowSelectColumn(...)`: Adds bulk-selection checkboxes when `BulkActions` are configured and `"admin:bulk_actions"` is permitted.

---

## Bulk Actions

Implement `admin.BulkAction` or instantiate `admin.BaseBulkAction` to add actions above the model list table:

```go
type BulkAction interface {
	Order() int
	Name() string
	Button() templ.Component
	HasPermission(r *http.Request, model *ModelDefinition) bool
	Execute(w http.ResponseWriter, r *http.Request, model *ModelDefinition, qs *queries.QuerySet[attrs.Definer]) (int, error)
}

type BaseBulkAction struct {
	ID            string
	Btn           templ.Component
	Ordering      int
	Permissions   []string
	PermissionsFn func(r *http.Request, model *ModelDefinition) bool
	Func          func(w http.ResponseWriter, r *http.Request, model *ModelDefinition, qs *queries.QuerySet[attrs.Definer]) (int, error)
}

func BulkActionButton(text any, typ components.ClassType, actionName string, requiresSelected bool) templ.Component

var BulkActionDelete *BaseBulkAction
```

- `qs` passed to `Execute` is already filtered to the selected primary keys (`<pk>__in`), or represents the unfiltered base queryset if no checkboxes were checked.
- Return `(changedCount, nil)` from `Execute`: if `changedCount > 0`, `ModelListHandler` flashes a success message and redirects back to the list view; if you redirect manually (as `BulkActionDelete` does to `admin:apps:model:bulk_delete?pk_list=...`), return `(0, nil)`.

---

## Model Startup Checks

`AdminApplication.Check` runs `ModelDefinition.Check` on every registered model and reports [Checks](../../checks.md) messages:

| Check ID | Level | Condition |
| :--- | :--- | :--- |
| `admin.model_not_fully_implemented` | Warning | Model does not implement both `models.ContextSaver` and `models.ContextDeleter` |
| `admin.models.add_view.panels` | Warning | `!DisallowCreate` and `AddView` has no `GetHandler`, `Fields`, `Exclude`, or `Panels` |
| `admin.models.edit_view.panels` | Warning | `!DisallowEdit` and `EditView` has no `GetHandler`, `Fields`, `Exclude`, or `Panels` |
| `admin.models.list_view.panels` | Warning | `ListView` has no `GetHandler`, `Columns`, `Fields`, or `Exclude` |
| `admin.models.formfields.add` | Warning | `!DisallowCreate` and `AddView` includes model fields whose `FormField()` returns `nil` |
| `admin.models.formfields.edit` | Warning | `!DisallowEdit` and form fields lack `FormField()` (note: `ModelDefinition.Check` currently inspects `m.AddView.ViewOptions` when checking `"edit"`) |
