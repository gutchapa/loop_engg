package catalog

import "testing"

func TestSearchByCategory(t *testing.T) {
	products := []Product{
		{ID: 1, Name: "Laptop", Category: "Electronics", Price: 999, Rating: 4.5, InStock: true},
		{ID: 2, Name: "Shirt", Category: "Clothing", Price: 29, Rating: 4.0, InStock: true},
		{ID: 3, Name: "Phone", Category: "Electronics", Price: 699, Rating: 4.8, InStock: true},
	}
	c := NewCatalog(products)
	results, total := c.Search(FilterOpts{Category: "Electronics"}, SortDefault, 1, 10)
	if total != 2 {
		t.Errorf("expected 2 electronics, got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestSearchByPriceRange(t *testing.T) {
	products := []Product{
		{ID: 1, Name: "Cheap", Price: 10, Rating: 3.0, InStock: true},
		{ID: 2, Name: "Mid", Price: 100, Rating: 4.0, InStock: true},
		{ID: 3, Name: "Expensive", Price: 1000, Rating: 5.0, InStock: true},
	}
	c := NewCatalog(products)
	results, total := c.Search(FilterOpts{MinPrice: 50, MaxPrice: 500}, SortDefault, 1, 10)
	if total != 1 || results[0].Name != "Mid" {
		t.Errorf("expected 1 mid-range product, got %d", total)
	}
}

func TestSearchSortByPrice(t *testing.T) {
	products := []Product{
		{ID: 1, Name: "A", Price: 300, Rating: 4.0, InStock: true},
		{ID: 2, Name: "B", Price: 100, Rating: 4.0, InStock: true},
		{ID: 3, Name: "C", Price: 200, Rating: 4.0, InStock: true},
	}
	c := NewCatalog(products)
	results, _ := c.Search(FilterOpts{}, SortPriceAsc, 1, 10)
	if results[0].Price != 100 || results[2].Price != 300 {
		t.Errorf("expected ascending prices")
	}

	results, _ = c.Search(FilterOpts{}, SortPriceDesc, 1, 10)
	if results[0].Price != 300 || results[2].Price != 100 {
		t.Errorf("expected descending prices")
	}
}

func TestSearchInStock(t *testing.T) {
	products := []Product{
		{ID: 1, Name: "In", Price: 10, Rating: 3.0, InStock: true},
		{ID: 2, Name: "Out", Price: 20, Rating: 3.0, InStock: false},
	}
	c := NewCatalog(products)
	_, total := c.Search(FilterOpts{InStock: true}, SortDefault, 1, 10)
	if total != 1 {
		t.Errorf("expected 1 in-stock, got %d", total)
	}
}

func TestSearchByName(t *testing.T) {
	products := []Product{
		{ID: 1, Name: "Wireless Mouse", Category: "Electronics", Price: 25, Rating: 4.0, InStock: true},
		{ID: 2, Name: "Mouse Pad", Category: "Accessories", Price: 10, Rating: 4.0, InStock: true},
	}
	c := NewCatalog(products)
	results, total := c.Search(FilterOpts{Search: "mouse"}, SortDefault, 1, 10)
	if total != 2 {
		t.Errorf("expected 2 mouse results, got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestPagination(t *testing.T) {
	products := make([]Product, 25)
	for i := range products {
		products[i] = Product{ID: i + 1, Price: float64(i), Rating: 4.0, InStock: true}
	}
	c := NewCatalog(products)

	page1, total := c.Search(FilterOpts{}, SortDefault, 1, 10)
	if len(page1) != 10 || total != 25 {
		t.Errorf("page1: len=%d total=%d", len(page1), total)
	}

	page3, _ := c.Search(FilterOpts{}, SortDefault, 3, 10)
	if len(page3) != 5 {
		t.Errorf("page3: expected 5, got %d", len(page3))
	}
}

func TestCategories(t *testing.T) {
	products := []Product{
		{Name: "A", Category: "X"},
		{Name: "B", Category: "Y"},
		{Name: "C", Category: "X"},
	}
	c := NewCatalog(products)
	cats := c.Categories()
	if len(cats) != 2 {
		t.Errorf("expected 2 categories, got %d", len(cats))
	}
}

func TestGenerateProducts(t *testing.T) {
	products := GenerateProducts(100)
	if len(products) != 100 {
		t.Errorf("expected 100, got %d", len(products))
	}
	if products[0].Name == "" || products[0].Category == "" {
		t.Error("generated product has empty fields")
	}
}

func BenchmarkSearchAll(b *testing.B) {
	products := GenerateProducts(10000)
	c := NewCatalog(products)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Search(FilterOpts{Category: "Electronics", InStock: true, MinRating: 3.0}, SortPriceAsc, 1, 20)
	}
}

func BenchmarkSortPriceDesc(b *testing.B) {
	products := GenerateProducts(10000)
	c := NewCatalog(products)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Search(FilterOpts{}, SortPriceDesc, 1, 20)
	}
}

func BenchmarkCategories(b *testing.B) {
	products := GenerateProducts(10000)
	c := NewCatalog(products)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Categories()
	}
}
