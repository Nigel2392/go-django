go-django (v2)
==============

**Django rewritten to Golang**

This code will be pushed to [go-django](https://github.com/Nigel2392/go-django) when ready.

This is a rewrite of the Django framework in Golang.

The goal is to provide a similar experience to Django, but with the performance of Golang.

There is a caveat though; we try to touch the database as little as possible.

This means that we don't have a full ORM like Django does.

Any database logic should be implemented by the end-developer.

## Not yet implemented

Some things which are not yet finished but have been planned.

This is not to say that these are the only things that will be implemented.

More features will be added as we go along.

- ~~Generic yet database- less page type~~ Page types (support for MySQL, SQLite and Postgres)
- [EditorJS](https://editorjs.io/) widget
- Auth application
- Adminsite application
- Full [wagtail](https://wagtail.org) block capabilities
