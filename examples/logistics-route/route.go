// Package route computes optimal delivery routes.
// Loop Engineering target: minimize route computation time (µs).
package route

import (
	"math"
	"sort"
)

type Point struct {
	ID int
	X, Y float64
}

// Route represents an ordered path through points.
type Route struct {
	Points   []Point
	Distance float64
}

// Distance between two points.
func Distance(a, b Point) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// TotalDistance computes the total route distance.
func TotalDistance(route []Point) float64 {
	d := 0.0
	for i := 1; i < len(route); i++ {
		d += Distance(route[i-1], route[i])
	}
	return d
}

// NearestNeighbor builds a route by always picking the closest unvisited point.
// O(n²) — good baseline for optimization.
func NearestNeighbor(points []Point) Route {
	if len(points) <= 1 {
		return Route{Points: points}
	}

	visited := make([]bool, len(points))
	route := make([]Point, 0, len(points))

	// Start at the first point
	current := 0
	route = append(route, points[current])
	visited[current] = true

	for len(route) < len(points) {
		nearest := -1
		nearestDist := math.MaxFloat64
		for j := 0; j < len(points); j++ {
			if visited[j] {
				continue
			}
			d := Distance(points[current], points[j])
			if d < nearestDist {
				nearestDist = d
				nearest = j
			}
		}
		current = nearest
		route = append(route, points[current])
		visited[current] = true
	}

	return Route{
		Points:   route,
		Distance: TotalDistance(route),
	}
}

// Greedy2Opt improves a route using 2-opt local search.
// Reverses segments to eliminate crossings.
func Greedy2Opt(route Route, iterations int) Route {
	best := make([]Point, len(route.Points))
	copy(best, route.Points)

	for iter := 0; iter < iterations; iter++ {
		improved := false
		for i := 0; i < len(best)-2; i++ {
			for j := i + 2; j < len(best)-1; j++ {
				oldDist := Distance(best[i], best[i+1]) + Distance(best[j], best[j+1])
				newDist := Distance(best[i], best[j]) + Distance(best[i+1], best[j+1])
				if newDist < oldDist {
					for l, r := i+1, j; l < r; l, r = l+1, r-1 {
						best[l], best[r] = best[r], best[l]
					}
					improved = true
				}
			}
		}
		if !improved {
			break
		}
	}

	return Route{
		Points:   best,
		Distance: TotalDistance(best),
	}
}

// GeneratePoints creates n random delivery points for benchmarking.
func GeneratePoints(n int) []Point {
	points := make([]Point, n)
	for i := range points {
		angle := 2 * math.Pi * float64(i) / float64(n)
		r := 1000.0
		points[i] = Point{
			ID: i + 1,
			X:  r * math.Cos(angle),
			Y:  r * math.Sin(angle),
		}
	}
	// Shuffle slightly for realism
	for i := range points {
		j := (i * 7) % n
		points[i], points[j] = points[j], points[i]
	}
	return points
}

// FurthestInsertion builds a route using farthest insertion heuristic.
// Generally better quality than nearest neighbor, slightly slower.
func FurthestInsertion(points []Point) Route {
	if len(points) <= 1 {
		return Route{Points: points}
	}

	visited := make([]bool, len(points))
	route := make([]Point, 0, len(points))

	// Start with the two farthest apart points
	farthestDist := 0.0
	farthestPair := [2]int{0, 1}
	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			d := Distance(points[i], points[j])
			if d > farthestDist {
				farthestDist = d
				farthestPair = [2]int{i, j}
			}
		}
	}

	route = append(route, points[farthestPair[0]], points[farthestPair[1]])
	visited[farthestPair[0]] = true
	visited[farthestPair[1]] = true

	for len(route) < len(points) {
		// Find the farthest unvisited point from any routed point
		farthestPt := -1
		farthestDist = 0
		for i := 0; i < len(points); i++ {
			if visited[i] {
				continue
			}
			minDist := math.MaxFloat64
			for _, rp := range route {
				d := Distance(points[i], rp)
				if d < minDist {
					minDist = d
				}
			}
			if minDist > farthestDist {
				farthestDist = minDist
				farthestPt = i
			}
		}

		// Insert at the position that minimizes route length
		bestPos := 1
		bestIncrease := math.MaxFloat64
		for pos := 1; pos < len(route); pos++ {
			increase := Distance(route[pos-1], points[farthestPt]) +
				Distance(points[farthestPt], route[pos]) -
				Distance(route[pos-1], route[pos])
			if increase < bestIncrease {
				bestIncrease = increase
				bestPos = pos
			}
		}

		newRoute := make([]Point, 0, len(route)+1)
		newRoute = append(newRoute, route[:bestPos]...)
		newRoute = append(newRoute, points[farthestPt])
		newRoute = append(newRoute, route[bestPos:]...)
		route = newRoute
		visited[farthestPt] = true
	}

	return Route{
		Points:   route,
		Distance: TotalDistance(route),
	}
}

// SortByClosestToOrigin returns points sorted by distance from (0,0).
func SortByClosestToOrigin(points []Point) Route {
	sorted := make([]Point, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool {
		return Distance(sorted[i], Point{}) < Distance(sorted[j], Point{})
	})
	return Route{
		Points:   sorted,
		Distance: TotalDistance(sorted),
	}
}
