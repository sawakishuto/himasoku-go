package auth

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
)

type FirebaseAuthClient struct {
	firebaseAuthClient *firebaseauth.Client
}

func NewFirebaseAuthClient() (Client, error) {
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create firebase app: %w", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create firebase auth client: %w", err)
	}
	return &FirebaseAuthClient{firebaseAuthClient: client}, nil
}

func (fc *FirebaseAuthClient) VerifyToken(ctx context.Context, rawToken string) (*Token, error) {
	token, err := fc.firebaseAuthClient.VerifyIDToken(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	return &Token{
		AuthTime: int64(token.AuthTime),
		Issuer:   string(token.Issuer),
		Audience: string(token.Audience),
		Expires:  int64(token.Expires),
		IssuedAt: int64(token.IssuedAt),
		Subject:  string(token.Subject),
		UID:      string(token.UID),
		Claims:   map[string]interface{}(token.Claims),
	}, nil
}
