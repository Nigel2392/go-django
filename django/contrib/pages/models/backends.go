package models

import models "github.com/Nigel2392/django/models"

var (
	registry   = models.NewBackendRegistry[Querier]()
	Register   = registry.RegisterForDriver
	GetBackend = registry.BackendForDB
)
