package hackernews

import (
	"context"
	"strconv"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	hn "github.com/hermanschaaf/hackernews"
	"github.com/mrusme/gobbs/models/author"
	"github.com/mrusme/gobbs/models/forum"
	"github.com/mrusme/gobbs/models/post"
	"github.com/mrusme/gobbs/models/reply"
	"github.com/mrusme/gobbs/system/adapter"
	"go.uber.org/zap"
)

type System struct {
	config map[string]interface{}
	logger *zap.SugaredLogger
	client *hn.Client
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

func (sys *System) GetCapabilities() []adapter.Capability {
	var caps []adapter.Capability

	caps = append(caps, adapter.Capability{
		ID:   "posts",
		Name: "Posts",
	})
	caps = append(caps, adapter.Capability{
		ID:   "groups",
		Name: "Groups",
	})
	caps = append(caps, adapter.Capability{
		ID:   "search",
		Name: "Search",
	})

	return caps
}

func (sys *System) Load() error {
	sys.client = hn.NewClient()
	return nil
}

func (sys *System) ListPosts(sysIdx int) ([]post.Post, error) {
	stories, err := sys.client.TopStories(context.Background())
	if err != nil {
		return []post.Post{}, err
	}

	converter := md.NewConverter("", true, nil)

	var models []post.Post
	for _, story := range stories[0:10] {
		i, err := sys.client.GetItem(context.Background(), story)
		if err != nil {
			sys.logger.Error(err)
			continue
		}

		t := "post"
		body := i.Text
		if i.URL != "" {
			t = "url"
			body = i.URL
		} else {
			bodyMd, err := converter.ConvertString(i.Text)
			if err == nil {
				body = bodyMd
			}
		}

		createdAt := time.Unix(int64(i.Time), 0)
		lastCommentedAt := createdAt

		var replies []reply.Reply
		for _, commentID := range i.Kids {
			replies = append(replies, reply.Reply{
				ID: strconv.Itoa(commentID),
			})
		}

		models = append(models, post.Post{
			ID: strconv.Itoa(i.ID),

			Subject: i.Title,
			Body:    body,

			Type: t,

			Pinned: false,
			Closed: i.Deleted,

			CreatedAt:       createdAt,
			LastCommentedAt: lastCommentedAt,

			Author: author.Author{
				ID:   i.By,
				Name: i.By,
			},

			Forum: forum.Forum{
				ID:   "new",
				Name: "New",
			},

			Replies: replies,

			SysIDX: sysIdx,
		})
	}

	return models, nil
}

func (sys *System) LoadPost(p *post.Post) error {
	return sys.loadReplies(&p.Replies)
}

func (sys *System) loadReplies(replies *[]reply.Reply) error {
	converter := md.NewConverter("", true, nil)
	for r := 0; r < len(*replies); r++ {
		re := &(*replies)[r]

		id, err := strconv.Atoi(re.ID)
		if err != nil {
			sys.logger.Error(err)
			continue
		}

		i, err := sys.client.GetItem(context.Background(), id)
		if err != nil {
			sys.logger.Error(err)
			continue
		}

		if i.Deleted || i.Dead {
			re.Deleted = true
		}

		createdAt := time.Unix(int64(i.Time), 0)

		re.Body = i.Text
		bodyMd, err := converter.ConvertString(i.Text)
		if err == nil {
			re.Body = bodyMd
		}

		re.CreatedAt = createdAt

		re.Author = author.Author{
			ID:   i.By,
			Name: i.By,
		}

		for _, commentID := range i.Kids {
			re.Replies = append(re.Replies, reply.Reply{
				ID: strconv.Itoa(commentID),
			})
		}

		if err := sys.loadReplies(&re.Replies); err != nil {
			sys.logger.Error(err)
		}
	}

	return nil
}