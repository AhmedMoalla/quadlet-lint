package main

func reverseMap[K, V comparable](input map[K]V) map[V]K {
	reversed := make(map[V]K, len(input))
	for k, v := range input {
		reversed[v] = k
	}
	return reversed
}
