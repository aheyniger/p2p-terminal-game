package tui

import "github.com/gdamore/tcell/v2"

var DefStyle = tcell.StyleDefault.Background(tcell.ColorDarkGreen).Foreground(tcell.ColorReset)
var BoxStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorPurple)
var NoBgStyle = tcell.StyleDefault.Foreground(tcell.ColorBlue).Background(tcell.ColorReset)
var NoFgStyle = tcell.StyleDefault.Foreground(tcell.ColorReset).Background(tcell.ColorGreen)
var FgStyle = tcell.StyleDefault.Foreground(tcell.ColorBlack)
var BgStyle = tcell.StyleDefault.Background(tcell.ColorWhite)
var HeaderStyle = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite)
