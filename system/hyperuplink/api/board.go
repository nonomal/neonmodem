package api

import (
	"context"
	"net/http"
)

const BoardBaseURL = "/"

type CategoryModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Position int    `json:"position"`
}

type ForumModel struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Position     int    `json:"position"`
	CategoryID   string `json:"category_id"`
	Description  string `json:"description"`
	CategoryName string `json:"category_name"`
	CategorySlug string `json:"category_slug"`
	Topics       int    `json:"topics"`
	Replies      int    `json:"replies"`
}

type CategoryWithForums struct {
	Category CategoryModel `json:"category"`
	Forums   []ForumModel  `json:"forums"`
}

type BoardResponse struct {
	CategoriesForums []CategoryWithForums `json:"categories_forums"`
	RecentTopics     []TopicModel         `json:"recent_topics"`
}

type BoardService interface {
	Get(
		ctx context.Context,
	) (*BoardResponse, error)
}

type BoardServiceHandler struct {
	client *Client
}

func (a *BoardServiceHandler) Get(
	ctx context.Context,
) (*BoardResponse, error) {
	req, err := a.client.NewRequest(ctx, http.MethodGet, BoardBaseURL, nil)
	if err != nil {
		return nil, err
	}

	response := new(BoardResponse)
	if err = a.client.Do(ctx, req, response); err != nil {
		return nil, err
	}

	return response, nil
}
