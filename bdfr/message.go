package bdfr

import (
	"encoding/json"
	"fmt"
	"os"
)

// Message is a reddit post or comment downloaded by bdfr.
type Message map[string]any

// GetString retrieves message's string value with the given key. It returns
// the empty string if the message does not contain the given key.
func (m Message) GetString(key string) string {
	st := m[key]
	if st == nil {
		return ""
	}
	return st.(string)
}

// SetString assigns the specified key-value pair to a message.
func (m Message) SetString(key string, val string) {
	m[key] = val
}

// GetSliceOfMessages retrieves message's value with the given key and returns
// it as a slice of messages. For example, it would retrieve a post's
// "comments" field. It returns nil if the message does not contain a matching
// key. It returns an error if the retrieved field is not a slice of messages.
func (m Message) GetSliceOfMessages(key string) ([]Message, error) {
	x := m[key]
	if x == nil {
		return nil, nil
	}

	slice, ok := x.([]any)
	if !ok {
		return nil, fmt.Errorf("wrong type for key=%s: have=%T want=[]any", key, x)
	}

	var ps []Message
	for i, a := range slice {
		m, ok := a.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("wrong type for key=%s,idx=%d: have=%T want=map[string]any", key, i, a)
		}
		ps = append(ps, Message(m))
	}

	return ps, nil
}

// ReadMessage unmarshals a bdfr message from disk.
func ReadMessage(filename string) (Message, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	m := Message{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}
