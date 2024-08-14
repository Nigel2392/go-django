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

This might take [a while](./docs/)...

- [Configuring](./docs/configuring.md)
- [Apps](./docs/apps.md)
- [Commands](./docs/commands.md)
- [Rendering](./docs/rendering.md)
- [Logging](./docs/logging.md)

## Not yet implemented

Some things which are not yet finished but have been planned.

This is not to say that these are the only things that will be implemented.

More features will be added as we go along.

- ~~Generic yet database- less page type~~ Page types (support for MySQL, SQLite and Postgres)
- [EditorJS](https://editorjs.io/) widget
- Auth application
  - Permissions
- Adminsite application
- Full [wagtail](https://wagtail.org) block capabilities
