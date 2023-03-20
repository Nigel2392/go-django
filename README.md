# Go-Django
A web framework built in Go, inspired by Django.  
This framework is still in development, and is not ready for production use.

## Information
This framework, is built from the ground up, and thus there is a lot of work to be done.  
It is not a port of Django, but rather a re-implementation of the core concepts of Django, in Go.  
The default ORM we use is [GORM](https://gorm.io/).  
This is because it is one of the most popular ORM's for Go, and it is very easy to use.

**Beware!**  
Most of the code is not tested thoroughly, and there could be bugs present.  
If you find any bugs, please report them in the github issues page.

## Finished:
- [X] Routing
- [X] Signals
- [X] Template file system/manager module
- [X] Media file system/manager module
- [X] middleware: CSRF protection, Sessions, AllowedHosts
- [X] Authentication
- [X] Messages (To the templates)
- [X] Sending emails
- [X] Secret keys
- [X] Admin panel extensions (Embed your own templates!)
- [X] Command-line flag package
- [X] Project-setup tool

## In progress:
- [ ] Admin panel
- [ ] Forms
- [ ] Middlewares (X-Frame-Options, others...)
- [ ] Testing
- [ ] Documentation
