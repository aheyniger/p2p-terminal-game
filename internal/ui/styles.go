package tui

import "github.com/gdamore/tcell/v2"

var ColorDarkSlateGray = tcell.ColorDarkSlateGray
var ColorWhite = tcell.ColorWhiteSmoke
var ColorBlack = tcell.ColorBlack
var ColorGray = tcell.ColorGray
var ColorBlue = tcell.ColorBlue

var DefaultBackgroundColor = ColorDarkSlateGray
var DefaultForegroundColor = ColorBlue
var HeaderColor = ColorWhite
var HeaderFieldColor = ColorBlack
var HeaderValueColor = ColorGray

var DefStyle = tcell.StyleDefault.Background(DefaultBackgroundColor).Foreground(DefaultBackgroundColor)
var BoxStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorPurple)

var NoBgStyle = DefStyle.Foreground(tcell.ColorBlue)

// var NoFgStyle = tcell.StyleDefault.Foreground(tcell.ColorReset).Background(tcell.ColorGreen)
// var FgStyle = tcell.StyleDefault.Foreground(tcell.ColorBlack)
// var BgStyle = tcell.StyleDefault.Background(tcell.ColorWhite)
var HeaderStyle = tcell.StyleDefault.Background(HeaderColor)
var HeaderFieldStyle = HeaderStyle.Foreground(HeaderFieldColor)
var HeaderValueStyle = HeaderStyle.Foreground(HeaderValueColor)
