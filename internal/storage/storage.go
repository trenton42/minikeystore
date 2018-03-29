package storage

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	glob "github.com/ryanuber/go-glob"
)

// Item holds a specific piece of data
// This piece of data can be of multiple types
type Item struct {
	stringValue string
	listValue   []string
	mapValue    map[string]string
	Type        string
}

// Storage holds all values in the keystore as well as acting as a mutex to protect read/write access to the data
type Storage struct {
	sync.RWMutex
	items map[string]*Item
	index sort.StringSlice
}

// New initializes a string store for use
func New() *Storage {
	var s Storage
	s.items = make(map[string]*Item)
	s.index = make([]string, 0)
	return &s
}

// checkkey does some basic error checking to see if a key exists and if it is the correct type
func (s *Storage) checkkey(key string, expectedType string, create bool) (*Item, error) {
	i, ok := s.items[key]
	if !ok {
		if !create {
			return nil, fmt.Errorf("key %s does not exist", key)
		}
		s.updateIndex(key, true)
		i = &Item{Type: expectedType}
		s.items[key] = i
		return i, nil
	}
	if i.Type != expectedType {
		return nil, fmt.Errorf("type %s is not %s", i.Type, expectedType)
	}
	return i, nil
}

// updateIndex sets and sorts the index whenever a key is added or removed from the collection
func (s *Storage) updateIndex(key string, add bool) {
	if add {
		i := s.index.Search(key)
		if i == len(s.index) {
			s.index = append(s.index, key)
			return
		}
		if s.index[i] == key {
			return
		}
		s.index = append(s.index[:i], append([]string{key}, s.index[i:]...)...)
	} else {
		i := s.index.Search(key)
		if i == len(s.index) {
			return
		}
		if s.index[i] != key {
			return
		}
		s.index = append(s.index[:i], s.index[i+1:]...)
	}
}

// Get returns an item as []byte
func (s *Storage) Get(key string) ([]byte, error) {
	s.RLock()
	defer s.RUnlock()
	i, ok := s.items[key]
	if !ok {
		return nil, fmt.Errorf("item does not exist")
	}
	switch i.Type {
	case "string":
		return json.Marshal(i.stringValue)
	case "list":
		return json.Marshal(i.listValue)
	case "map":
		return json.Marshal(i.mapValue)
	}
	return nil, fmt.Errorf("unknown type")
}

// Set puts an item into storage
func (s *Storage) Set(key string, value interface{}) error {
	s.Lock()
	defer s.Unlock()
	var i Item
	switch v := value.(type) {
	case string:
		i.Type = "string"
		i.stringValue = v
	case []string:
		i.Type = "list"
		i.listValue = v
	case map[string]string:
		i.Type = "map"
		i.mapValue = v
	case []interface{}:
		i.Type = "list"
		i.listValue = make([]string, len(v))
		for index, val := range v {
			i.listValue[index] = fmt.Sprintf("%s", val)
		}
	case map[string]interface{}:
		i.Type = "map"
		i.mapValue = make(map[string]string)
		for key, val := range v {
			i.mapValue[key] = fmt.Sprintf("%s", val)
		}
	default:
		return fmt.Errorf("unknown type")
	}
	if _, ok := s.items[key]; !ok {
		s.updateIndex(key, true)
	}
	s.items[key] = &i
	return nil
}

// Delete removes a value stored at key
func (s *Storage) Delete(key string) {
	s.Lock()
	if _, ok := s.items[key]; ok {
		s.updateIndex(key, false)
	}
	delete(s.items, key)
	s.Unlock()
}

// Append pushes a value on the end of a list, or returns error if the type is not a list
func (s *Storage) Append(key string, value string) error {
	s.Lock()
	defer s.Unlock()
	i, err := s.checkkey(key, "list", true)
	if err != nil {
		return err
	}
	if i.listValue == nil {
		i.listValue = make([]string, 0)
	}
	i.listValue = append(i.listValue, value)
	s.items[key] = i
	return nil
}

// Pop removes a value from the end of a list and returns it, or returns error if the type is not a list or the list is empty
func (s *Storage) Pop(key string) (string, error) {
	s.Lock()
	defer s.Unlock()
	i, err := s.checkkey(key, "list", false)
	if err != nil {
		return "", err
	}
	if len(i.listValue) == 0 {
		return "", fmt.Errorf("list is empty")
	}
	var value string
	value, i.listValue = i.listValue[len(i.listValue)-1], i.listValue[:len(i.listValue)-1]
	return value, nil
}

// MapGet returns a specific key from a map type
func (s *Storage) MapGet(key string, mkey string) (string, error) {
	s.RLock()
	defer s.RUnlock()
	i, err := s.checkkey(key, "map", false)
	if err != nil {
		return "", err
	}
	if i.mapValue == nil {
		return "", fmt.Errorf("map key %s does not exists", mkey)
	}

	if val, ok := i.mapValue[mkey]; ok {
		return val, nil
	}
	return "", fmt.Errorf("map key %s does not exists", mkey)
}

// MapSet sets a key on a map type
func (s *Storage) MapSet(key string, mkey string, value string) error {
	s.Lock()
	defer s.Unlock()
	i, err := s.checkkey(key, "map", true)
	if err != nil {
		return err
	}
	if i.mapValue == nil {
		i.mapValue = make(map[string]string)
	}
	i.mapValue[mkey] = value
	s.items[key] = i
	return nil
}

// MapDelete removes a key from a map type
func (s *Storage) MapDelete(key string, mkey string) error {
	s.Lock()
	defer s.Unlock()
	i, err := s.checkkey(key, "map", false)
	if err != nil {
		return err
	}
	if i.mapValue == nil {
		i.mapValue = make(map[string]string)
	}
	delete(i.mapValue, mkey)
	s.items[key] = i
	return nil
}

// GetIndex returns a slice of the index based on a globbed key.
func (s *Storage) GetIndex(search string) []string {
	// Match everything, so return full index
	if search == "*" {
		return s.index
	}
	parts := strings.Split(search, "*")
	// There is no wildcard at all, so only return the index key (if it exists)
	if len(parts) == 1 {
		index := s.index.Search(search)
		if s.index[index] != search {
			return make([]string, 0)
		}
		return []string{s.index[index]}
	}
	// Now we are going in for a full glob search
	result := make([]string, 0)
	for _, val := range s.index {
		if glob.Glob(search, val) {
			result = append(result, val)
		}
	}
	return result
}
