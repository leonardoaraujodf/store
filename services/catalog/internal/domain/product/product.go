package product

import "errors"

var ErrEmptyID = errors.New("product id cannot be empty")
var ErrEmptyName = errors.New("product name cannot be empty")
var ErrNegativePrice = errors.New("product price cannot be negative")
var ErrInvalidCurrency = errors.New("invalid currency name")

type Product struct {
	ID              string
	Name            string
	Description     string
	PriceMinorUnits int64
	Currency        string
}

func New(id string, name string, description string, priceMinorUnits int64, currency string) (Product, error) {
	if id == "" {
		return Product{}, ErrEmptyID
	}
	if name == "" {
		return Product{}, ErrEmptyName
	}
	if priceMinorUnits < 0 {
		return Product{}, ErrNegativePrice
	}

	if len(currency) != 3 {
		return Product{}, ErrInvalidCurrency
	}

	for _, c := range currency {
		if c < 'A' || c > 'Z' {
			return Product{}, ErrInvalidCurrency
		}
	}

	return Product{
		ID:              id,
		Name:            name,
		Description:     description,
		PriceMinorUnits: priceMinorUnits,
		Currency:        currency,
	}, nil
}
