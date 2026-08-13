package api

import (
	"context"
	"net/http"
)

const SessionBaseURL = "/session"

type UserModel struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type SessionResponse struct {
	User UserModel `json:"user"`
}

type SessionService interface {
	Whoami(
		ctx context.Context,
	) (*SessionResponse, error)
}

type SessionServiceHandler struct {
	client *Client
}

func (a *SessionServiceHandler) Whoami(
	ctx context.Context,
) (*SessionResponse, error) {
	req, err := a.client.NewRequest(ctx, http.MethodGet, SessionBaseURL, nil)
	if err != nil {
		return nil, err
	}

	response := new(SessionResponse)
	if err = a.client.Do(ctx, req, response); err != nil {
		return nil, err
	}

	return response, nil
}
