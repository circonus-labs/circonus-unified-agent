package filestack

import "strconv"

type Event struct {
	Action    string `json:"action"`
	TimeStamp int64  `json:"timestamp"`
	ID        int    `json:"id"`
}

func (fe *Event) Tags() map[string]string {
	return map[string]string{
		"action": fe.Action,
	}
}

func (fe *Event) Fields() map[string]interface{} {
	return map[string]interface{}{
		"id": strconv.Itoa(fe.ID),
	}
}
