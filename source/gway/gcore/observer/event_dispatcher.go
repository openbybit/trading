package observer

type EventDispatcher interface {
	Listenable
	// Dispatch event
	Dispatch(event Event) error
}
