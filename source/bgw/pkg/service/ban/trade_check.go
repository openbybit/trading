package ban

type key struct {
	biz   string
	tag   string
	value string
}

func newBanKey(biz, tag, value string) key {
	return key{
		biz:   biz,
		tag:   tag,
		value: value,
	}
}
