package database

import "sync"

type UserRecord struct {
	FirebaseUID     string
	Email           string
	PlaidAccessToken string
}

type Store struct {
	mu    sync.RWMutex
	users map[string]UserRecord
}

var DefaultStore = &Store{users: make(map[string]UserRecord)}

func InitializeStore() {
	DefaultStore.mu.Lock()
	defer DefaultStore.mu.Unlock()
	if DefaultStore.users == nil {
		DefaultStore.users = make(map[string]UserRecord)
	}
}

func (s *Store) SaveUser(record UserRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[record.FirebaseUID] = record
}

func (s *Store) GetUser(firebaseUID string) (UserRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[firebaseUID]
	return user, ok
}

func (s *Store) UpdatePlaidAccessToken(firebaseUID, accessToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[firebaseUID]
	if !ok {
		user = UserRecord{FirebaseUID: firebaseUID}
	}
	user.PlaidAccessToken = accessToken
	s.users[firebaseUID] = user
	return nil
}
