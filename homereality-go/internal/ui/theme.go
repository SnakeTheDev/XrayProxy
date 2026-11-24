package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Colors matched to the Windows panel.
var (
	ColBg     = color.NRGBA{R: 0x16, G: 0x18, B: 0x1D, A: 0xFF}
	ColCard   = color.NRGBA{R: 0x22, G: 0x26, B: 0x30, A: 0xFF}
	ColText   = color.NRGBA{R: 0xE8, G: 0xEA, B: 0xED, A: 0xFF}
	ColMuted  = color.NRGBA{R: 0x8A, G: 0x90, B: 0x9C, A: 0xFF}
	ColGreen  = color.NRGBA{R: 0x34, G: 0xC7, B: 0x59, A: 0xFF}
	ColRed    = color.NRGBA{R: 0xFF, G: 0x45, B: 0x3A, A: 0xFF}
	ColAmber  = color.NRGBA{R: 0xFF, G: 0x9F, B: 0x0A, A: 0xFF}
	ColOff    = color.NRGBA{R: 0x3A, G: 0x3F, B: 0x4B, A: 0xFF}
	ColBorder = color.NRGBA{R: 0x2C, G: 0x31, B: 0x3C, A: 0xFF}
)

type hrTheme struct{}

func (h *hrTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameBackground, theme.ColorNameOverlayBackground:
		return ColBg
	case theme.ColorNameButton, theme.ColorNameInputBackground:
		return ColCard
	case theme.ColorNameForeground, theme.ColorNamePlaceHolder:
		return ColText
	case theme.ColorNameDisabled:
		return ColMuted
	case theme.ColorNamePrimary, theme.ColorNameSuccess:
		return ColGreen
	case theme.ColorNameError:
		return ColRed
	case theme.ColorNameHover:
		return ColOff
	case theme.ColorNameSeparator, theme.ColorNameShadow:
		return ColBorder
	case theme.ColorNameMenuBackground, theme.ColorNameHeaderBackground:
		return ColCard
	default:
		return theme.DefaultTheme().Color(n, v)
	}
}

func (h *hrTheme) Font(s fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(s)
}

func (h *hrTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (h *hrTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInlineIcon:
		return 20
	default:
		return theme.DefaultTheme().Size(n)
	}
}

func NewTheme() fyne.Theme { return &hrTheme{} }
