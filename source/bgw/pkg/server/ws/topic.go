package ws

// Topics is topics collection
type Topics struct {
	items map[string]struct{}
}

func newTopics() Topics {
	return Topics{items: make(map[string]struct{})}
}

// Size get topics size
func (t Topics) Size() int {
	return len(t.items)
}

// Values is topics collection
func (t Topics) Values() []string {
	ss := make([]string, 0, t.Size())
	for k := range t.items {
		ss = append(ss, k)
	}

	return ss
}

// Clear clear all topics
func (t *Topics) Clear() {
	t.items = make(map[string]struct{})
}

// Add add topic
func (t *Topics) Add(items ...string) []string {
	res := make([]string, 0, len(items))
	for _, s := range items {
		if _, ok := t.items[s]; !ok {
			t.items[s] = struct{}{}
			res = append(res, s)
		}
	}

	return res
}

// Remove remove topic
func (t *Topics) Remove(items ...string) []string {
	res := make([]string, 0, len(items))
	for _, s := range items {
		if _, ok := t.items[s]; ok {
			delete(t.items, s)
			res = append(res, s)
		}
	}

	return res
}

// Merge merge other topics
func (t *Topics) Merge(other Topics) {
	for k := range other.items {
		t.items[k] = struct{}{}
	}
}

// Contains check if the topic is in the collection
func (t Topics) Contains(topic string) bool {
	_, ok := t.items[topic]
	return ok
}

// ContainsAny check if any of the topics is in the collection
func (t Topics) ContainsAny(topics []string) bool {
	for _, v := range topics {
		_, ok := t.items[v]
		if ok {
			return true
		}
	}

	return false
}

func (t Topics) Clone() Topics {
	res := Topics{items: make(map[string]struct{})}
	for k, v := range t.items {
		res.items[k] = v
	}
	return res
}
