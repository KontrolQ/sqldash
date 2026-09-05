package collections

type OrderedMap[K comparable, V any] struct {
	keys   []K
	values map[K]V
}

func OrderedMapOf[K comparable, V any]() OrderedMap[K, V] {
	return OrderedMap[K, V]{
		keys:   make([]K, 0),
		values: make(map[K]V),
	}
}

func (self *OrderedMap[K, V]) Set(key K, value V) {
	if _, exists := self.values[key]; !exists {
		self.keys = append(self.keys, key)
	}
	self.values[key] = value
}

func (self *OrderedMap[K, V]) Get(key K) (V, bool) {
	value, exists := self.values[key]
	return value, exists
}

func (self *OrderedMap[K, V]) All() []V {
	result := make([]V, 0, len(self.keys))
	for _, key := range self.keys {
		result = append(result, self.values[key])
	}
	return result
}

func (self *OrderedMap[K, V]) Len() int {
	return len(self.keys)
}
