package features

import (
	"context"

	"github.com/Nigel2392/go-django/src/contrib/editor"
)

var (
	_ editor.PrefetchableFeature      = (*PrefetchableFeature)(nil)
	_ editor.PrefetchableFeatureBlock = (*PrefetchableFeatureBlock)(nil)
)

type PrefetchableFeature struct {
	Block
	Prefetch func(ctx context.Context, data []editor.BlockData) (map[string]editor.BlockData, error)
}

func (f *PrefetchableFeature) PrefetchData(ctx context.Context, data []editor.BlockData) (map[string]editor.BlockData, error) {
	if f.Prefetch != nil {
		return f.Prefetch(ctx, data)
	}
	return nil, nil
}

func (f *PrefetchableFeature) Render(data editor.BlockData) editor.FeatureBlock {
	var block = f.Block.Render(data).(*FeatureBlock)
	block.FeatureObject = f
	return &PrefetchableFeatureBlock{
		FeatureBlock: block,
	}
}

type PrefetchableFeatureBlock struct {
	*FeatureBlock
}

func (b *PrefetchableFeatureBlock) WithData(ctx context.Context, data editor.BlockData) {
	b.FeatureBlock.FeatureData = data
}
