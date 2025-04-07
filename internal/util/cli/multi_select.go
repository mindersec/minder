// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MultiSelect implements the necessary logic to implement an
// interactive multi-select menu for the CLI.
//
// Given a list of string as choices, returns those interactively
// selected by the user.
func MultiSelect(choices []string) ([]string, error) {
	items := make([]list.Item, 0, len(choices))
	for _, c := range choices {
		items = append(items, item{title: c})
	}

	slices.SortFunc(items, func(a, b list.Item) int {
		return cmp.Compare(a.(item).title, b.(item).title)
	})

	l := list.New(items, itemDelegate{}, 0, 0)
	l.Title = "Select repos to register"
	l.Styles.Title = Header
	l.AdditionalShortHelpKeys = extraKeys
	l.AdditionalFullHelpKeys = extraKeys

	height := 30 // 20 + 10, 10 is a magic number to show 20 items
	if size := len(items); size < 20 {
		height = size + 10
	}
	model := model{list: l, height: height}
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return nil, err
	}

	selection := make([]string, 0, len(items))
	for _, listItem := range items {
		item := listItem.(item)
		if item.checked {
			selection = append(selection, item.title)
		}
	}

	return selection, nil
}

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(SecondaryColor)
)

// item represents the model object for every item in the multi-select
// list.
type item struct {
	title   string
	checked bool
}

func (i item) Title() string       { return i.title }
func (item) Description() string   { return "" }
func (i item) FilterValue() string { return i.title }

// itemDelegate packs all the logic related to rendering items in TUI
type itemDelegate struct{}

func (itemDelegate) Height() int                             { return 1 }
func (itemDelegate) Spacing() int                            { return 0 }
func (itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	checked := "[ ]"
	if i.checked {
		checked = "[x]"
	}

	fmt.Fprint(w, fn(checked, i.title))
}

// model represents the actual multi-select, with all its rendering,
// navigation, and selection logic.
type model struct {
	list   list.Model
	height int
}

func (model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(0, m.height)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "enter":
			return m, tea.Quit
		case " ":
			idx := m.list.Index()
			oldItem := m.list.SelectedItem().(item)
			cmd := m.list.SetItem(idx, item{
				title:   oldItem.title,
				checked: !oldItem.checked,
			})
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return m.list.View()
}

func extraKeys() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("space"),
			key.WithHelp("space", "select item"),
		),
	}
}
