package auth

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

func NewFirebaseAuthClient() (*firebaseauth.Client, error) {
	ctx := context.Background()

	opt := option.WithCredentialsFile("path/to/credentials.json")

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create firebase app: %w", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create firebase auth client: %w", err)
	}
	return client, nil
}
