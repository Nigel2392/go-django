package orders

import (
	"encoding/json"

	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
)

type ProductData struct {
	Product  *products.Product
	Specific products.ProductModel
}

func (d *ProductData) UnmarshalJSON(data []byte) error {
	var snapshot struct {
		Product  json.RawMessage
		Specific json.RawMessage
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}
	if snapshot.Product == nil && snapshot.Specific == nil {
		// Older order items stored the base product directly.
		snapshot.Product = data
	}

	var decoded ProductData
	if len(snapshot.Product) != 0 {
		if err := json.Unmarshal(snapshot.Product, &decoded.Product); err != nil {
			return err
		}
	}
	if len(snapshot.Specific) != 0 && string(snapshot.Specific) != "null" {
		model, err := newSnapshotProduct(decoded.Product)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(snapshot.Specific, model); err != nil {
			return err
		}
		decoded.Specific = model
	}
	*d = decoded
	return nil
}

func newSnapshotProduct(product *products.Product) (products.ProductModel, error) {
	if product == nil || product.SpecificContentType == nil {
		return nil, errors.CorruptedData.Wrap("specific product content type is missing")
	}
	model, ok := product.SpecificContentType.New().(products.ProductModel)
	if !ok {
		return nil, errors.CorruptedData.Wrapf("content type %q is not a product model", product.SpecificContentType)
	}
	return model, nil
}
