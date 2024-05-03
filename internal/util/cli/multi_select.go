//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	choices  []string         // available choices
	cursor   int              // which item our cursor is pointing at
	selected map[int]struct{} // which items are selected
}

// MultiSelect implements the necessary logic to implement an
// interactive multi-select menu for the CLI.
//
// Given a list of string as choices, returns those interactively
// selected by the user.
func MultiSelect(choices []string) ([]string, error) {
	model := model{
		choices:  choices,
		selected: make(map[int]struct{}),
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return nil, err
	}

	selection := make([]string, 0, len(model.selected))
	for i := range model.selected {
		selection = append(selection, model.choices[i])
	}

	return selection, nil
}

func (_ model) Init() tea.Cmd {
	return nil
}

//nolint:revive // this seems to me like a false positive
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "enter", "q": // <C-c>, <enter> and <q> quits
			return m, tea.Quit
		case "up": // <up> and <down> arrows move through selections
			if m.cursor > 0 {
				m.cursor--
			}
		case "down": // <up> and <down> arrows move through selections
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "space", " ": // <space> selects entries
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	s := "Select repositories to register with Minder:\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	return s
}
