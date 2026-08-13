package postshow

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/mrusme/neonmodem/models/post"
	"github.com/mrusme/neonmodem/models/reply"
	"github.com/mrusme/neonmodem/system/lib"
	"github.com/mrusme/neonmodem/ui/ctx"
)

func (m Model) View() string {
	return m.tk.View(&m, true)
}

func buildView(mi interface{}, cached bool) string {
	var m *Model = mi.(*Model)

	if vcache := m.tk.DefaultCaching(cached); vcache != "" {
		return vcache
	}

	return m.tk.Dialog(
		"Post",
		viewportStyle.Render(m.viewport.View()),
		true,
	)
}

type renderedPost struct {
	content    string
	replyIDs   []string
	allReplies []*reply.Reply
}

func writesOrAsks(subject string) string {
	if strings.HasSuffix(subject, "?") {
		return "asks"
	}
	return "writes"
}

func renderLoadingPlaceholder(c *ctx.Ctx, p *post.Post) string {
	return fmt.Sprintf(
		" %s\n\n %s\n\n %s\n",
		c.Theme.Post.Author.Render(
			fmt.Sprintf("%s %s:", p.Author.Name, writesOrAsks(p.Subject)),
		),
		c.Theme.Post.Subject.Render(p.Subject),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("#777777")).
			Render("Loading post, please wait (press esc to go back) ..."),
	)
}

func renderPost(
	c *ctx.Ctx,
	p *post.Post,
	viewportWidth int,
	imageWidth int,
	isCurrent func() bool,
) (rendered renderedPost, ok bool) {
	style := styles.LightStyle
	if c.DarkBackground {
		style = styles.DarkStyle
	}

	glam, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(viewportWidth),
	)
	if err != nil {
		c.Logger.Error(err)
		glam = nil
	}

	render := func(s string) string {
		if glam == nil {
			return s
		}
		out, rerr := glam.Render(s)
		if rerr != nil {
			c.Logger.Error(rerr)
			return s
		}
		return out
	}

	body := render(p.Body)
	if c.Config.RenderImages {
		body = lib.RenderInlineImages(c, body, imageWidth)
	}

	if !isCurrent() {
		return renderedPost{}, false
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf(
		" %s\n\n %s\n%s",
		c.Theme.Post.Author.Render(
			fmt.Sprintf("%s %s:", p.Author.Name, writesOrAsks(p.Subject)),
		),
		c.Theme.Post.Subject.Render(p.Subject),
		body,
	))

	rendered.replyIDs = []string{p.ID}
	rendered.allReplies = []*reply.Reply{}

	caps := (*c.Systems[p.SysIDX]).GetCapabilities()
	if !caps.IsCapableOf("list:replies") {
		rendered.content = out.String()
		return rendered, true
	}

	if p.CurrentRepliesStartIDX > 0 {
		out.WriteString(render(
			"\n---\nOlder replies available, press `z` to load\n\n---\n"))
	}

	var walk func(inReplyTo string, replies *[]reply.Reply) bool
	walk = func(inReplyTo string, replies *[]reply.Reply) bool {
		if replies == nil {
			return true
		}

		for ri := range *replies {
			if !isCurrent() {
				return false
			}

			re := &(*replies)[ri]

			var body string
			var authorName string
			if re.Deleted {
				body = "\n  DELETED\n\n"
				authorName = "DELETED"
			} else {
				body = render(re.Body)
				authorName = re.Author.Name
			}

			rendered.replyIDs = append(rendered.replyIDs, re.ID)
			rendered.allReplies = append(rendered.allReplies, re)
			idx := len(rendered.replyIDs) - 1

			replyIdPadding := viewportWidth - len(authorName) - len(inReplyTo) - 28
			if replyIdPadding < 0 {
				replyIdPadding = 0
			}

			out.WriteString(fmt.Sprintf(
				"\n\n %s %s%s%s\n%s",
				c.Theme.Reply.Author.Render(authorName),
				lipgloss.NewStyle().
					Foreground(c.Theme.Reply.Author.GetBackground()).
					Render(fmt.Sprintf("writes in reply to %s:", inReplyTo)),
				strings.Repeat(" ", replyIdPadding),
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("#777777")).
					Render(fmt.Sprintf("#%d", idx)),
				body,
			))

			if !walk(re.Author.Name, &re.Replies) {
				return false
			}
		}

		return true
	}

	if !walk(p.Author.Name, &p.Replies) {
		return renderedPost{}, false
	}

	rendered.content = out.String()
	return rendered, true
}
