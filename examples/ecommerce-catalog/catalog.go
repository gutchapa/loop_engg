// Package catalog manages an e-commerce product catalog with faceted search.
// Loop Engineering target: minimize filter+sort latency (µs).
package catalog

import (
	"math"
	"sort"
	"strings"
)

type Product struct {
	ID       int
	Name     string
	Category string
	Price    float64
	Rating   float64
	InStock  bool
	Tags     []string
}

type Catalog struct {
	products []Product
}

// NewCatalog creates a catalog from products.
func NewCatalog(products []Product) *Catalog {
	return &Catalog{products: products}
}

// FilterOpts captures all filtering criteria.
type FilterOpts struct {
	Category string
	MinPrice float64
	MaxPrice float64
	MinRating float64
	InStock  bool // true = only in-stock
	Search   string
}

// SortOrder for results.
type SortOrder int

const (
	SortDefault SortOrder = iota
	SortPriceAsc
	SortPriceDesc
	SortRatingDesc
	SortNameAsc
)

// Search returns filtered and sorted products matching the criteria.
func (c *Catalog) Search(opts FilterOpts, sortBy SortOrder, page, pageSize int) ([]Product, int) {
	// Filter
	var filtered []Product
	for _, p := range c.products {
		if opts.Category != "" && p.Category != opts.Category {
			continue
		}
		if p.Price < opts.MinPrice || (opts.MaxPrice > 0 && p.Price > opts.MaxPrice) {
			continue
		}
		if p.Rating < opts.MinRating {
			continue
		}
		if opts.InStock && !p.InStock {
			continue
		}
		if opts.Search != "" && !strings.Contains(strings.ToLower(p.Name), strings.ToLower(opts.Search)) {
			continue
		}
		filtered = append(filtered, p)
	}

	total := len(filtered)

	// Sort
	switch sortBy {
	case SortPriceAsc:
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Price < filtered[j].Price })
	case SortPriceDesc:
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Price > filtered[j].Price })
	case SortRatingDesc:
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Rating > filtered[j].Rating })
	case SortNameAsc:
		sort.Slice(filtered, func(i, j int) bool {
			return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
		})
	}

	// Paginate
	if page < 1 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start >= len(filtered) {
		return nil, total
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total
}

// Categories returns all unique categories.
func (c *Catalog) Categories() []string {
	seen := make(map[string]bool)
	for _, p := range c.products {
		seen[p.Category] = true
	}
	var cats []string
	for cat := range seen {
		cats = append(cats, cat)
	}
	sort.Strings(cats)
	return cats
}

// GenerateProducts creates n sample products for benchmarking.
func GenerateProducts(n int) []Product {
	categories := []string{"Electronics", "Clothing", "Books", "Home", "Sports", "Food", "Toys", "Beauty"}
	tags := []string{"new", "sale", "popular", "limited", "eco", "premium"}
	adjectives := []string{"Premium", "Basic", "Pro", "Lite", "Ultra", "Classic", "Modern", "Eco"}
	nouns := []string{"Widget", "Gadget", "Item", "Tool", "Kit", "Set", "Pack", "Bundle"}

	products := make([]Product, n)
	for i := range products {
		products[i] = Product{
			ID:       i + 1,
			Name:     adjectives[i%len(adjectives)] + " " + nouns[i%len(nouns)],
			Category: categories[i%len(categories)],
			Price:    math.Round((float64(i%9900)+100)*100) / 100,
			Rating:   math.Round((1+float64(i%40)/10)*10) / 10,
			InStock:  i%3 != 0,
			Tags:     []string{tags[i%len(tags)]},
		}
	}
	return products
}
