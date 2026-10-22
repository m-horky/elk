package product

import (
	"fmt"

	"github.com/m-horky/elk/internal/fs"
	internalproducts "github.com/m-horky/elk/internal/rhsmx509/product"
	"github.com/m-horky/elk/pkg/constants"
)

// Product contains the metadata of one installed Red Hat product.
type Product = internalproducts.Product

// Get returns products found in the system's Red Hat product certificates.
func Get() ([]Product, error) {
	products, err := internalproducts.Read(
		fs.Filesystem{},
		constants.Path(constants.DefaultProductDir),
		constants.Path(constants.ProductDir),
	)
	if err != nil {
		return nil, fmt.Errorf("load product: %w", err)
	}

	return products, nil
}
