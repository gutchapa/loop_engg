package route

import (
	"math"
	"testing"
)

func TestDistance(t *testing.T) {
	d := Distance(Point{X: 0, Y: 0}, Point{X: 3, Y: 4})
	if math.Abs(d-5.0) > 0.0001 {
		t.Errorf("expected 5, got %f", d)
	}
}

func TestTotalDistance(t *testing.T) {
	points := []Point{
		{X: 0, Y: 0},
		{X: 3, Y: 0},
		{X: 3, Y: 4},
	}
	d := TotalDistance(points)
	if math.Abs(d-7.0) > 0.0001 {
		t.Errorf("expected 7, got %f", d)
	}
}

func TestNearestNeighbor(t *testing.T) {
	points := GeneratePoints(20)
	route := NearestNeighbor(points)
	if len(route.Points) != 20 {
		t.Errorf("expected 20 points, got %d", len(route.Points))
	}
	if route.Distance <= 0 {
		t.Error("distance should be positive")
	}
}

func TestNearestNeighborSingle(t *testing.T) {
	route := NearestNeighbor([]Point{{ID: 1, X: 0, Y: 0}})
	if len(route.Points) != 1 {
		t.Error("single point should return itself")
	}
}

func TestGreedy2Opt(t *testing.T) {
	points := GeneratePoints(10)
	route := NearestNeighbor(points)
	improved := Greedy2Opt(route, 10)
	if improved.Distance > route.Distance {
		t.Errorf("2-opt should not increase distance: %.2f > %.2f", improved.Distance, route.Distance)
	}
}

func TestFurthestInsertion(t *testing.T) {
	points := GeneratePoints(20)
	route := FurthestInsertion(points)
	if len(route.Points) != 20 {
		t.Errorf("expected 20 points, got %d", len(route.Points))
	}
	if route.Distance <= 0 {
		t.Error("distance should be positive")
	}
}

func TestSortByClosest(t *testing.T) {
	points := []Point{
		{X: 100, Y: 0},
		{X: 5, Y: 5},
		{X: 0, Y: 0},
		{X: 10, Y: 0},
	}
	route := SortByClosestToOrigin(points)
	if route.Points[0].X != 0 || route.Points[0].Y != 0 {
		t.Error("first point should be origin")
	}
}

func TestQualityComparison(t *testing.T) {
	// NN vs FI should both produce valid routes; FI is generally better
	points := GeneratePoints(10)
	nn := NearestNeighbor(points)
	fi := FurthestInsertion(points)
	if nn.Distance <= 0 || fi.Distance <= 0 {
		t.Error("both methods should produce positive distances")
	}
}

func BenchmarkNearestNeighbor(b *testing.B) {
	points := GeneratePoints(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NearestNeighbor(points)
	}
}

func BenchmarkFurthestInsertion(b *testing.B) {
	points := GeneratePoints(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FurthestInsertion(points)
	}
}

func BenchmarkGreedy2Opt(b *testing.B) {
	points := GeneratePoints(50)
	route := NearestNeighbor(points)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Greedy2Opt(route, 10)
	}
}

func BenchmarkSortByClosest(b *testing.B) {
	points := GeneratePoints(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SortByClosestToOrigin(points)
	}
}

func BenchmarkDistance(b *testing.B) {
	p1 := Point{X: 1234.567, Y: 7890.123}
	p2 := Point{X: 4321.789, Y: 9876.543}
	for i := 0; i < b.N; i++ {
		Distance(p1, p2)
	}
}
