// Package search indexes and searches patient records.
// Loop Engineering target: minimize search latency (ms).
package search

import (
	"sort"
	"strings"
)

type Patient struct {
	ID        int
	Name      string
	Diagnosis string
	Age       int
	Ward      string
}

// Index holds searchable patient data.
type Index struct {
	patients []Patient
}

// NewIndex creates a search index from patient records.
func NewIndex(patients []Patient) *Index {
	return &Index{patients: patients}
}

// SearchByName returns patients whose name contains the query (case-insensitive).
func (idx *Index) SearchByName(query string) []Patient {
	q := strings.ToLower(query)
	var results []Patient
	for _, p := range idx.patients {
		if strings.Contains(strings.ToLower(p.Name), q) {
			results = append(results, p)
		}
	}
	return results
}

// SearchByDiagnosis returns patients matching a diagnosis keyword.
func (idx *Index) SearchByDiagnosis(query string) []Patient {
	q := strings.ToLower(query)
	var results []Patient
	for _, p := range idx.patients {
		if strings.Contains(strings.ToLower(p.Diagnosis), q) {
			results = append(results, p)
		}
	}
	return results
}

// FilterByAge returns patients within an age range [min, max].
func (idx *Index) FilterByAge(min, max int) []Patient {
	var results []Patient
	for _, p := range idx.patients {
		if p.Age >= min && p.Age <= max {
			results = append(results, p)
		}
	}
	return results
}

// SortByName returns patients sorted alphabetically by name.
func SortByName(patients []Patient) []Patient {
	sorted := make([]Patient, len(patients))
	copy(sorted, patients)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})
	return sorted
}

// GeneratePatients creates n sample patient records for benchmarking.
func GeneratePatients(n int) []Patient {
	diagnoses := []string{"Hypertension", "Diabetes", "Asthma", "Cancer", "Pneumonia", "Fracture", "Covid-19", "Stroke"}
	wards := []string{"ICU", "General", "Maternity", "Pediatrics", "Cardiology", "Oncology"}
	firstNames := []string{"Alice", "Bob", "Carol", "Dave", "Eve", "Frank", "Grace", "Hank", "Iris", "Jack"}
	lastNames := []string{"Smith", "Jones", "Brown", "Taylor", "Wilson", "Lee", "Kim", "Patel", "Chen", "Gupta"}

	patients := make([]Patient, n)
	for i := range patients {
		patients[i] = Patient{
			ID:        i + 1,
			Name:      firstNames[i%len(firstNames)] + " " + lastNames[i%len(lastNames)],
			Diagnosis: diagnoses[i%len(diagnoses)],
			Age:       20 + (i % 60),
			Ward:      wards[i%len(wards)],
		}
	}
	return patients
}
