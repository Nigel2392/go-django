# Go-Django
A web framework built in Go, inspired by Django.
This framework is still in development, and is not ready for production use.

## Information
This framework, is built from the ground up, and thus there is a lot of work to be done.  
It is not a port of Django, but rather a re-implementation of the core concepts of Django, in Go.  
The default ORM we use is [GORM](https://gorm.io/).  
This is because it one of the most popular ORM's for Go, and it is very easy to use.

**Beware!**  
Most of the code is not tested thoroughly, and there could be bugs present.  
If you find any bugs, please report them in the github issues page.

## Finished:
- [x] Routing
- [x] Signals
- [x] Template file system/manager module
- [x] Media file system/manager module
- [x] middleware: CSRF protection, Sessions, AllowedHosts
- [x] Authentication
- [x] Messages (To the templates)
- [x] Sending emails
- [x] Secret keys
- [x] Admin panel extensions (Embed your own templates!)

## In progress:
- [ ] Admin panel
- [ ] Forms
- [ ] Other middlewares (X-Frame-Options, others...)
- [ ] Testing
- [ ] Documentation
