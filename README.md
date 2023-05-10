# Go-Django

A web framework built in Go, inspired by Django.
This framework is still in development, and is not ready for production use.

It might still undergo many reworks and breaking changes.

## Information

This framework, is built from the ground up, and thus there is a lot of work to be done.
It is not a port of Django, but rather a re-implementation of the core concepts of Django, in Go.
~~The default ORM we use is [GORM](https://gorm.io/).
This is because it is one of the most popular ORM's for Go, and it is very easy to use.~~

The framework works through interfaces, thus you must add your own implementation for saving models, etc.

**Beware!**
Most of the code is not tested thoroughly, and there could be bugs present.
If you find any bugs, please report them in the github issues page.

## Rework:

The framework, and mainly the admin site used to have a dependency on GORM.

This has been removed, and it is now up to the Developer to implement a way to save your models.

The admin package has went through a complete rework. To see a full implementation of models in the admin site; have a look at the following files:

- auth/user.go
- auth/groups.go
- auth/permissions.go

It is now also possible to create custom fields for the admin-site forms. 


The auth package used to also depend on gorm. This has now also been removed for generated queries by SQLC.

## Recommended installation process

Make sure you have your GOPATH set up correctly.

### Installing the project-setup tool

```bash
go install github.com/Nigel2392/go-django/go-django@vX.X.X.X
```

This will install the `go-django` command-line tool, which you can use to create a new project.

### Creating a new project

```bash
go-django -startproject <project-name>
```

This will create a new project in the current directory, with the name you specified.
Following that, change into the project directory, make sure you install the latest tag.
Our tags are in the format of `vX.X.X.X`, which might make the go tool struggle.
After changing into the project directory, run the following command:

```bash
go get github.com/Nigel2392/go-django@vX.X.X.X
go mod tidy
```

This will install the latest version of the framework, and update the go.mod file.
Now you can run the project with the following command:

```bash
go run ./src/
```

### Adminsite

In the main file are a few developer credentials registered.
These are used to log into the adminsite.
You can view the adminsite by going to the `<<DOMAIN>>/admin/` url (As specified in config.go).

## Finished:

- [X] Routing
- [X] Signals
- [X] Template file system/manager module
- [X] ~~Media file system/manager module~~ (Reworked)
- [X] middleware: CSRF protection, Sessions, AllowedHosts
- [X] *Authentication* (Reworked)
- [X] Messages (To the templates)
- [X] Sending emails
- [X] ~~Admin panel extensions (Embed your own templates!)~~ (Removed after rework, WIP)
- [X] ~~Secret keys, secure hashing with secret.Hash and secret.Compare~~ (Repackaged into separate github repository)
- [X] Command-line flag package
- [X] Project-setup tool
- [X] Debug recovery page middleware (Only when running with -debug flag)

## In progress:

- [ ] ~~Forms~~ (Repackaged into separate github repository.)
- [ ] Testing
- [ ] Documentation
