package images

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/goldcrest"
)

var _ Querier = (*imageQuerier)(nil)

type Querier interface {
	InsertImage(ctx context.Context, img *Image) error
	UpdateImage(ctx context.Context, img *Image) error
	DeleteImage(ctx context.Context, img *Image) error

	SelectByID(ctx context.Context, id uint32) (*Image, error)
	SelectByPath(ctx context.Context, path string) (*Image, error)
	SelectByFileHash(ctx context.Context, fileHash string) (*Image, error)

	SelectBasic(ctx context.Context, limit, offset int32) ([]*Image, error)
	SelectLargeToSmall(ctx context.Context, limit, offset int32) ([]*Image, error)
	SelectNewestToOldest(ctx context.Context, limit, offset int32) ([]*Image, error)
	SelectOldestToNewest(ctx context.Context, limit, offset int32) ([]*Image, error)
	SelectSmallToLarge(ctx context.Context, limit, offset int32) ([]*Image, error)
}

type imageQuerier struct {
	*queries
}

func runHooks(hookName string, img *Image) error {
	var hooks = goldcrest.Get[ImageHook](hookName)
	for _, hook := range hooks {
		err := hook(img)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewQueryset(db DBTX) Querier {
	if app == nil {
		assert.Fail("images app not initialized")
	}

	return &imageQuerier{
		queries: newQueries(db),
	}
}

func (q *imageQuerier) InsertImage(ctx context.Context, img *Image) error {
	var err = runHooks(HOOK_BEFORE_IMAGE_CREATE, img)
	if err != nil {
		return err
	}

	id, err := q.queries.InsertImage(ctx,
		img.Title,
		img.Path,
		img.CreatedAt,
		img.FileSize,
		img.FileHash,
	)
	if err != nil {
		return err
	}

	img.ID = uint32(id)

	return runHooks(HOOK_AFTER_IMAGE_CREATE, img)
}

func (q *imageQuerier) UpdateImage(ctx context.Context, img *Image) error {
	var err = runHooks(HOOK_BEFORE_IMAGE_UPDATE, img)
	if err != nil {
		return err
	}

	err = q.queries.UpdateImage(ctx,
		img.Title,
		img.Path,
		img.CreatedAt,
		img.FileSize,
		img.FileHash,
		img.ID,
	)

	if err != nil {
		return err
	}

	return runHooks(HOOK_AFTER_IMAGE_UPDATE, img)
}

func (q *imageQuerier) DeleteImage(ctx context.Context, img *Image) error {
	var err = runHooks(HOOK_BEFORE_IMAGE_DELETE, img)
	if err != nil {
		return err
	}

	err = q.queries.DeleteImage(ctx, img.ID)
	if err != nil {
		return err
	}

	return runHooks(HOOK_AFTER_IMAGE_DELETE, img)
}
