package popuplist

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mrusme/neonmodem/ui/cmd"
)

func handleSelect(mi interface{}) (bool, []tea.Cmd) {
	var m *Model = mi.(*Model)
	var cmds []tea.Cmd

	cmds = append(cmds, cmd.New(
		cmd.WMCloseWin,
		WIN_ID,
		cmd.Arg{Name: "selectionID", Value: m.selectionID},
		cmd.Arg{Name: "selected", Value: m.list.SelectedItem()},
	).Tea())
	return true, cmds
}

func handleViewResize(mi interface{}) (bool, []tea.Cmd) {
	var m *Model = mi.(*Model)
	var cmds []tea.Cmd

	listWidth := m.tk.ViewWidth() - 2
	listHeight := m.tk.ViewHeight() - 1

	m.ctx.Theme.PopupList.List.Focused.Width(listWidth)
	m.ctx.Theme.PopupList.List.Blurred.Width(listWidth)
	m.ctx.Theme.PopupList.List.Focused.Height(listHeight)
	m.ctx.Theme.PopupList.List.Blurred.Height(listHeight)
	m.list.SetSize(
		listWidth-2,
		listHeight-2,
	)

	return false, cmds
}

func handleWinOpenCmd(mi interface{}, c cmd.Command) (bool, []tea.Cmd) {
	var m *Model = mi.(*Model)
	var cmds []tea.Cmd

	if c.Target == WIN_ID {
		m.selectionID = c.GetArg("selectionID").(string)
		m.items = c.GetArg("items").([]list.Item)
		m.list.SetItems(m.items)
		return true, cmds
	}

	return false, cmds
}

func handleWinFreshDataCmd(mi interface{}, c cmd.Command) (bool, []tea.Cmd) {
	var m *Model = mi.(*Model)
	var cmds []tea.Cmd

	if c.Target != WIN_ID {
		return false, cmds
	}

	if items, ok := c.GetArg("items").([]list.Item); ok {
		m.items = items
		m.list.SetItems(m.items)
	}
	m.ctx.Loading = false

	if errs, ok := c.GetArg("errors").([]error); ok && len(errs) > 0 {
		for _, err := range errs {
			m.ctx.Logger.Error(err)
		}
		cmds = append(cmds, cmd.New(
			cmd.MsgError,
			"*",
			cmd.Arg{Name: "errors", Value: errs},
		).Tea())
	}

	return true, cmds
}
