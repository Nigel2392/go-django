package images

const (
	HOOK_BEFORE_IMAGE_CREATE = "images:before:image:create"
	HOOK_AFTER_IMAGE_CREATE  = "images:after:image:create"

	HOOK_BEFORE_IMAGE_UPDATE = "images:before:image:update"
	HOOK_AFTER_IMAGE_UPDATE  = "images:after:image:update"

	HOOK_BEFORE_IMAGE_DELETE = "images:before:image:delete"
	HOOK_AFTER_IMAGE_DELETE  = "images:after:image:delete"
)

type ImageHook func(image *Image) error
