package models

import models "github.com/Nigel2392/go-django/src/models"

var (
	registry   = models.NewBackendRegistry[Querier]()
	Register   = registry.RegisterForDriver
	GetBackend = registry.BackendForDB
)
