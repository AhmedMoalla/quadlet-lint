package utils

func MapSlice[S any, D any](source []S, mapFn func(S) D) []D {
	result := make([]D, len(source))
	for i, value := range source {
		result[i] = mapFn(value)
	}

	return result
}
