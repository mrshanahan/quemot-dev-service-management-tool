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

func Values[TKey comparable, TValue any](m map[TKey]TValue) []TValue {
	vs := make([]TValue, len(m))
	i := 0
	for _, v := range m {
		vs[i] = v
		i++
	}
	return vs
}

func Filter[V any](xs []V, pred func(x V) bool) []V {
	ys := []V{}
	for _, x := range xs {
		if pred(x) {
			ys = append(ys, x)
		}
	}
	return ys
}

func Map[S any, T any](xs []S, f func(S) T) []T {
	ys := make([]T, len(xs))
	for i := range xs {
		ys[i] = f(xs[i])
	}
	return ys
}
