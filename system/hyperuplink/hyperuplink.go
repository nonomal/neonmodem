package hyperuplink

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/araddon/dateparse"
	"github.com/mrusme/neonmodem/models/author"
	"github.com/mrusme/neonmodem/models/forum"
	"github.com/mrusme/neonmodem/models/post"
	"github.com/mrusme/neonmodem/models/reply"
	"github.com/mrusme/neonmodem/system/adapter"
	"github.com/mrusme/neonmodem/system/hyperuplink/api"
	"go.uber.org/zap"
)

type System struct {
	ID        int
	config    map[string]interface{}
	logger    *zap.SugaredLogger
	client    *api.Client
	clientCfg api.ClientConfig
}

func (sys *System) GetID() int {
	return sys.ID
}

func (sys *System) SetID(id int) {
	sys.ID = id
}

func (sys *System) GetConfig() map[string]interface{} {
	return sys.config
}

func (sys *System) SetConfig(cfg *map[string]interface{}) {
	sys.config = *cfg
}

func (sys *System) SetLogger(logger *zap.SugaredLogger) {
	sys.logger = logger
}

func (sys *System) GetCapabilities() adapter.Capabilities {
	var caps []adapter.Capability

	caps = append(caps,
		adapter.Capability{
			ID:   "connect:multiple",
			Name: "Connect Multiple",
		},
		adapter.Capability{
			ID:   "list:forums",
			Name: "List Forums",
		},
		adapter.Capability{
			ID:   "list:posts",
			Name: "List Posts",
		},
		adapter.Capability{
			ID:   "create:post",
			Name: "Create Post",
		},
		adapter.Capability{
			ID:   "list:replies",
			Name: "List Replies",
		},
		adapter.Capability{
			ID:   "create:reply",
			Name: "Create Reply",
		},
	)

	return caps
}

func (sys *System) FilterValue() string {
	return fmt.Sprintf(
		"Hyperuplink %s",
		sys.config["url"],
	)
}

func (sys *System) Title() string {
	sysUrl := sys.config["url"].(string)
	u, err := url.Parse(sysUrl)
	if err != nil {
		return sysUrl
	}

	return u.Hostname()
}

func (sys *System) Description() string {
	return fmt.Sprintf(
		"Hyperuplink",
	)
}

func (sys *System) Load() error {
	u := sys.config["url"]
	if u == nil {
		return nil
	}

	credentials := make(map[string]string)
	for k, v := range (sys.config["credentials"]).(map[string]interface{}) {
		credentials[k] = v.(string)
	}

	proxy := ""
	if p, ok := sys.config["proxy"].(string); ok {
		proxy = p
	}

	sys.clientCfg = api.NewDefaultClientConfig(
		u.(string),
		proxy,
		credentials["token"],
		sys.logger,
	)
	sys.client = api.NewClient(&sys.clientCfg)

	return nil
}

func (sys *System) ListForums() ([]forum.Forum, error) {
	board, err := sys.client.Board.Get(context.Background())
	if err != nil {
		return []forum.Forum{}, err
	}

	var models []forum.Forum
	for _, cf := range board.CategoriesForums {
		for _, f := range cf.Forums {
			models = append(models, forum.Forum{
				ID:   f.ID,
				Name: fmt.Sprintf("%s/%s", cf.Category.Name, f.Name),

				Info: f.Description,

				SysIDX: sys.ID,
			})
		}
	}

	return models, nil
}

func (sys *System) ListPosts(forumID string) ([]post.Post, error) {
	resp, err := sys.client.Topics.List(context.Background(), forumID, 1)
	if err != nil {
		return []post.Post{}, err
	}

	baseURL := sys.config["url"].(string)

	var models []post.Post
	for _, t := range resp.Topics {
		models = append(models, sys.topicToPost(&t, baseURL))
	}

	return models, nil
}

func (sys *System) topicToPost(t *api.TopicModel, baseURL string) post.Post {
	createdAt, err := dateparse.ParseAny(t.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	lastCommentedAt := createdAt
	if t.LastReplyAt != "" {
		if parsed, perr := dateparse.ParseAny(t.LastReplyAt); perr == nil {
			lastCommentedAt = parsed
		}
	}

	return post.Post{
		ID: t.ID,

		Subject: t.Name,
		Body:    t.Text,

		Type: "post",

		Pinned: t.Pinned,
		Closed: t.LockedAt != "",

		CreatedAt:       createdAt,
		LastCommentedAt: lastCommentedAt,

		Author: author.Author{
			ID:   t.AuthorID,
			Name: t.AuthorUsername,
		},

		Forum: forum.Forum{
			ID:   t.ForumID,
			Name: fmt.Sprintf("%s/%s", t.CategoryName, t.ForumName),

			SysIDX: sys.ID,
		},

		TotalReplies:           t.Replies,
		CurrentRepliesStartIDX: -1,

		URL: fmt.Sprintf("%s/_%s/%s/%s",
			baseURL, t.CategorySlug, t.ForumSlug, t.Slug),

		SysIDX: sys.ID,
	}
}

func (sys *System) LoadPost(p *post.Post) error {
	var allReplies []api.ReplyModel

	page := 1
	for {
		resp, err := sys.client.Topics.Show(context.Background(), p.ID, page)
		if err != nil {
			return err
		}

		if page == 1 {
			p.Body = resp.Topic.Text
			p.Pinned = resp.Topic.Pinned
			p.Closed = resp.Topic.LockedAt != ""
		}

		allReplies = append(allReplies, resp.Replies...)

		if resp.Pages <= page {
			break
		}
		page++
	}

	p.Replies = sys.buildReplyTree(p.ID, allReplies)
	p.TotalReplies = len(allReplies)
	p.CurrentRepliesStartIDX = -1

	return nil
}

func (sys *System) buildReplyTree(
	topicID string,
	replies []api.ReplyModel,
) []reply.Reply {
	present := make(map[string]bool, len(replies))
	for _, r := range replies {
		present[r.ID] = true
	}

	childrenOf := make(map[string][]api.ReplyModel)
	for _, r := range replies {
		parent := r.ReplyID
		if parent != "" && !present[parent] {
			parent = ""
		}
		childrenOf[parent] = append(childrenOf[parent], r)
	}

	var build func(parentID string) []reply.Reply
	build = func(parentID string) []reply.Reply {
		out := []reply.Reply{}
		for _, r := range childrenOf[parentID] {
			createdAt, err := dateparse.ParseAny(r.CreatedAt)
			if err != nil {
				createdAt = time.Now()
			}

			out = append(out, reply.Reply{
				ID:        r.ID,
				InReplyTo: topicID,

				Body: r.Text,

				Deleted: r.DeletedAt != "",

				CreatedAt: createdAt,

				Author: author.Author{
					ID:   r.AuthorID,
					Name: r.AuthorUsername,
				},

				Replies: build(r.ID),

				SysIDX: sys.ID,
			})
		}
		return out
	}

	return build("")
}

func (sys *System) CreatePost(p *post.Post) error {
	created, err := sys.client.Posts.Create(context.Background(), &api.NewPostModel{
		Name:    p.Subject,
		Text:    p.Body,
		ForumID: p.Forum.ID,
		Kind:    "regular",
	})
	if err != nil {
		return err
	}

	p.ID = created.ID
	return nil
}

func (sys *System) CreateReply(r *reply.Reply) error {
	var topicID string
	var body api.CreateReplyModel

	if r.InReplyTo == "" {
		topicID = r.ID
	} else {
		topicID = r.InReplyTo
		body.ReplyID = r.ID
	}
	body.Text = r.Body

	created, err := sys.client.Topics.CreateReply(
		context.Background(),
		topicID,
		&body,
	)
	if err != nil {
		return err
	}

	r.ID = created.ID
	return nil
}
