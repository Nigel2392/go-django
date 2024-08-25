go-django (v2)
==============

![1719351174099](.github/image/README/1719351174099.png)

**Django rewritten to Golang**

This code will be pushed to [go-django](https://github.com/Nigel2392/go-django) when ready.

This is a rewrite of the Django framework in Golang.

The goal is to provide a similar experience to Django, but with the performance of Golang.

At the core this is meant to be a web framework, but it also includes sub-packages to create a CMS-like experience.

There is a caveat though; we try to touch the database as little as possible.

This means that we don't have a full ORM like Django does.

Any database logic should be implemented by the end-developer, but some sub-packages do provide backends to use with MySQL and SQLite. Postgres is not planned yet.

## Docs:

- [Configuring your server](./docs/configuring.md)
- [Creating an app](./docs/apps.md)
- [Creating Forms](./docs/forms/forms.md)
  - [Working with Fields](./docs/forms/fields.md)
  - [Working with Widgets](./docs/forms/widgets.md)
  - [Passing and creating Media](./docs/forms/media.md)
- [Defining your models](./docs/attrs.md)
- [Setting up Routing](./docs/routing.md)
- [Creating Management Commands](./docs/commands.md)
- [Rendering Your Templates](./docs/rendering.md)
- [Usage of Contenttypes](./docs/contenttypes.md)
- [Setting up and calling Hooks](./docs/hooks.md)
- [Working with the Filesystem](./docs/filesystem.md)
- [Serving your Staticfiles](./docs/staticfiles.md)
- [Setting up Logging](./docs/logging.md)

### Contrib apps:

- [sessions](./docs/apps/sessions.md)
- [admin](./docs/apps/admin)
- [auditlogs](./docs/apps/auditlogs.md)
- [auth](./docs/apps/auth)
- [pages](./docs/apps/pages)
- [editorjs](./docs/apps/editor.md)

## Help Needed:

- [ ] Block application:
  - [ ] Javascript for structblock
  - [ ] Javascript for listblock
  - [ ] (maybe) Javascript for fieldblock

