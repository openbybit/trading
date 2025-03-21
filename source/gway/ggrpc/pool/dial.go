package pool

var Dial DialFunc

func SetDialFunc(f DialFunc) {
	Dial = f
}
