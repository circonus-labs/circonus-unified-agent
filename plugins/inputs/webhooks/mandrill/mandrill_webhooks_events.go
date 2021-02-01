package mandrill

// type Event interface {
// 	Tags() map[string]string
// 	Fields() map[string]interface{}
// }

type Event struct {
	EventName string `json:"event"`
	TimeStamp int64  `json:"ts"`
	ID        string `json:"_id"`
}

func (me *Event) Tags() map[string]string {
	return map[string]string{
		"event": me.EventName,
	}
}

func (me *Event) Fields() map[string]interface{} {
	return map[string]interface{}{
		"id": me.ID,
	}
}
