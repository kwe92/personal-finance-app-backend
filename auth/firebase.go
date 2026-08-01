package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type VerifiedFirebaseUser struct {
	UID   string
	Email string
}

var firebaseApp *firebase.App

var ErrFirebaseNotConfigured = errors.New("firebase credentials are not configured")

func initFirebaseApp() error {
	if firebaseApp != nil {
		return nil
	}

	ctx := context.Background()
	var opts []option.ClientOption

	if path := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); path != "" {
		opts = append(opts, option.WithCredentialsFile(path))
	}
	if raw := os.Getenv("FIREBASE_CREDENTIALS_JSON"); raw != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(raw)))
	}

	app, err := firebase.NewApp(ctx, nil, opts...)
	if err != nil {
		return err
	}
	firebaseApp = app
	return nil
}

func VerifyIDToken(ctx context.Context, rawToken string) (*VerifiedFirebaseUser, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, fmt.Errorf("missing id token")
	}

	if err := initFirebaseApp(); err != nil {
		return nil, ErrFirebaseNotConfigured
	}

	client, err := firebaseApp.Auth(ctx)
	if err != nil {
		return nil, err
	}

	decoded, err := client.VerifyIDToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	email, _ := decoded.Claims["email"].(string)
	return &VerifiedFirebaseUser{UID: decoded.UID, Email: email}, nil
}

func VerifyIDTokenWithFallback(ctx context.Context, rawToken string, fallbackUID string, fallbackEmail string) (*VerifiedFirebaseUser, error) {
	if strings.TrimSpace(rawToken) == "" {
		return &VerifiedFirebaseUser{UID: fallbackUID, Email: fallbackEmail}, nil
	}
	return VerifyIDToken(ctx, rawToken)
}

func GetFirebaseApp() (*firebase.App, error) {
	if err := initFirebaseApp(); err != nil {
		return nil, ErrFirebaseNotConfigured
	}
	return firebaseApp, nil
}

func NewAuthClient() (*firebaseauth.Client, error) {
	if err := initFirebaseApp(); err != nil {
		return nil, ErrFirebaseNotConfigured
	}
	return firebaseApp.Auth(context.Background())
}
