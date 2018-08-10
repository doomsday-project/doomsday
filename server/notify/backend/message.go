package backend

type Message []MessagePart

type MessagePart interface {
	//This is a hack just to limit what is a "MessagePart" at compilation time
	IsMessagePart()
}

type MText struct {
	Text string
}

func (MText) IsMessagePart() {}

type MLink struct {
	Link string
	Text string
}

func (MLink) IsMessagePart() {}
