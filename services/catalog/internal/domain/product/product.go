package product

type Product struct {
	ID              string
	Name            string
	Description     string
	PriceMinorUnits int
	Currency        string
}

func New(id string, name string, description string, priceMinorUnits int, currency string) (Product, error) {
	return Product{
		ID:              id,
		Name:            name,
		Description:     description,
		PriceMinorUnits: priceMinorUnits,
		Currency:        currency,
	}, nil
}
