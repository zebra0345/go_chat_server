package model

type Message struct {
	Type     string `json:"type"`
	Nickname string `json:"nickname,omitempty"`
	Message  string `json:"message,omitempty"`
}