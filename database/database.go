package database

import (
	"context"
	"errors"
	"log"

	"personal_finance_backend/auth"

	"cloud.google.com/go/firestore"
)

type UserRecord struct {
	FirebaseUID      string `firestore:"firebaseUID"`
	Email            string `firestore:"email"`
	PlaidAccessToken string `firestore:"plaidAccessToken"`
	IsPlaidLinked    bool   `firestore:"is_plaid_linked"`
}

type Store struct {
	firestoreClient *firestore.Client
}

var DefaultStore = &Store{}

func InitializeStore() {
	if DefaultStore.firestoreClient != nil {
		return
	}

	app, err := auth.GetFirebaseApp()
	if err != nil {
		log.Println("warning: unable to initialize Firebase app for Firestore persistence:", err)
		return
	}

	ctx := context.Background()
	client, err := app.Firestore(ctx)
	if err != nil {
		log.Println("warning: unable to initialize Firestore client:", err)
		return
	}

	DefaultStore.firestoreClient = client
}

func (s *Store) GetUser(firebaseUID string) (UserRecord, bool) {
	if s.firestoreClient == nil {
		return UserRecord{}, false
	}

	ctx := context.Background()
	doc, err := s.firestoreClient.Collection("users").Doc(firebaseUID).Get(ctx)
	if err != nil || !doc.Exists() {
		return UserRecord{}, false
	}

	var user UserRecord
	if err := doc.DataTo(&user); err != nil {
		log.Println("warning: Firestore GetUser decode failed:", err)
		return UserRecord{}, false
	}

	return user, true
}

func (s *Store) UpdatePlaidAccessToken(firebaseUID, accessToken string) error {
	if s.firestoreClient == nil {
		return errors.New("firestore client not initialized")
	}

	ctx := context.Background()
	_, err := s.firestoreClient.Collection("users").Doc(firebaseUID).Update(ctx, []firestore.Update{
		{Path: "plaidAccessToken", Value: accessToken},
		{Path: "is_plaid_linked", Value: true},
	})
	return err
}
