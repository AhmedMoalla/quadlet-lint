package main

func reverseMap[K, V comparable](input map[K]V) map[V]K {
	reversed := make(map[V]K, len(input))
	for k, v := range input {
		reversed[v] = k
	}
	return reversed
}

func mergeMaps[K comparable, V any](m1 map[K]V, m2 map[K]V) map[K]V {
	merged := make(map[K]V, len(m1)+len(m2))
	for key, value := range m1 {
		merged[key] = value
	}
	for key, value := range m2 {
		merged[key] = value
	}
	return merged
}
