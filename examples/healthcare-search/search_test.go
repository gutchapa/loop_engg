package search

import (
	"testing"
)

var testPatients = []Patient{
	{ID: 1, Name: "Alice Smith", Diagnosis: "Hypertension", Age: 45, Ward: "Cardiology"},
	{ID: 2, Name: "Bob Jones", Diagnosis: "Diabetes", Age: 62, Ward: "General"},
	{ID: 3, Name: "Carol Brown", Diagnosis: "Asthma", Age: 28, Ward: "General"},
	{ID: 4, Name: "Dave Wilson", Diagnosis: "Cancer", Age: 55, Ward: "Oncology"},
	{ID: 5, Name: "Eve Taylor", Diagnosis: "Pneumonia", Age: 70, Ward: "ICU"},
	{ID: 6, Name: "Frank Lee", Diagnosis: "Hypertension", Age: 35, Ward: "Cardiology"},
	{ID: 7, Name: "Grace Kim", Diagnosis: "Diabetes", Age: 50, Ward: "General"},
	{ID: 8, Name: "Hank Patel", Diagnosis: "Fracture", Age: 22, Ward: "Orthopedics"},
}

func TestSearchByName(t *testing.T) {
	idx := NewIndex(testPatients)
	results := idx.SearchByName("alice")
	if len(results) != 1 || results[0].ID != 1 {
		t.Errorf("expected 1 result (Alice), got %d", len(results))
	}
}

func TestSearchByNamePartial(t *testing.T) {
	idx := NewIndex(testPatients)
	results := idx.SearchByName("sm")
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestSearchByNameCaseInsensitive(t *testing.T) {
	idx := NewIndex(testPatients)
	r1 := idx.SearchByName("eve")
	r2 := idx.SearchByName("EVE")
	if len(r1) != 1 || len(r2) != 1 {
		t.Error("case-insensitive search failed")
	}
}

func TestSearchByDiagnosis(t *testing.T) {
	idx := NewIndex(testPatients)
	results := idx.SearchByDiagnosis("hypertension")
	if len(results) != 2 {
		t.Errorf("expected 2 hypertension patients, got %d", len(results))
	}
}

func TestSearchByDiagnosisPartial(t *testing.T) {
	idx := NewIndex(testPatients)
	results := idx.SearchByDiagnosis("diab")
	if len(results) != 2 {
		t.Errorf("expected 2 diabetes patients, got %d", len(results))
	}
}

func TestFilterByAge(t *testing.T) {
	idx := NewIndex(testPatients)
	results := idx.FilterByAge(30, 50)
	for _, p := range results {
		if p.Age < 30 || p.Age > 50 {
			t.Errorf("patient %d age %d outside range", p.ID, p.Age)
		}
	}
}

func TestSortByName(t *testing.T) {
	sorted := SortByName(testPatients)
	if len(sorted) != len(testPatients) {
		t.Fatal("length mismatch")
	}
	for i := 1; i < len(sorted); i++ {
		if sorted[i].Name < sorted[i-1].Name {
			t.Errorf("not sorted: %s > %s", sorted[i-1].Name, sorted[i].Name)
		}
	}
}

func TestSearchNoResults(t *testing.T) {
	idx := NewIndex(testPatients)
	results := idx.SearchByName("zzz")
	if len(results) != 0 {
		t.Error("expected no results")
	}
}

func TestGeneratePatients(t *testing.T) {
	patients := GeneratePatients(100)
	if len(patients) != 100 {
		t.Errorf("expected 100 patients, got %d", len(patients))
	}
	if patients[0].Name == "" || patients[0].Diagnosis == "" {
		t.Error("generated patient has empty fields")
	}
}

func BenchmarkSearchByName(b *testing.B) {
	patients := GeneratePatients(1000)
	idx := NewIndex(patients)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.SearchByName("smith")
	}
}

func BenchmarkSearchByDiagnosis(b *testing.B) {
	patients := GeneratePatients(1000)
	idx := NewIndex(patients)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.SearchByDiagnosis("cancer")
	}
}

func BenchmarkFilterByAge(b *testing.B) {
	patients := GeneratePatients(1000)
	idx := NewIndex(patients)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.FilterByAge(20, 40)
	}
}

func BenchmarkSortByName(b *testing.B) {
	patients := GeneratePatients(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SortByName(patients)
	}
}
