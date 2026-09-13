package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

func main() {
	_ = runewidth.StringWidth("hmx")

	s, err := tcell.NewScreen()
	if err != nil {
		panic(err)
	}
	if err := s.Init(); err != nil {
		panic(err)
	}
	defer s.Fini()

	style := tcell.StyleDefault
	for i, r := range "hmx" {
		s.SetContent(i, 0, r, nil, style)
	}
	s.Show()

	for {
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if ev.Rune() == 'q' || ev.Key() == tcell.KeyCtrlC {
				return
			}
		}
	}
}
