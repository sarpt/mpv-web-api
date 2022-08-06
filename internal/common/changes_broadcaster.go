package common

type ChangeVariant string

type Change interface {
	Variant() ChangeVariant
}

type ChangesSubscriber[CT Change] interface {
	Receive(change CT)
}

type ChangesBroadcaster[CT Change] struct {
	*Broadcaster[CT]
}

func NewChangesBroadcaster[CT Change]() *ChangesBroadcaster[CT] {
	return &ChangesBroadcaster[CT]{
		NewBroadcaster[CT](),
	}
}
