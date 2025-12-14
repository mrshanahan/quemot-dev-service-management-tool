package utils

func Keys[TKey comparable, TValue any](m map[TKey]TValue) []TKey {
	ks := make([]TKey, len(m))
	i := 0
	for k := range m {
		ks[i] = k
		i++
	}
	return ks
}
