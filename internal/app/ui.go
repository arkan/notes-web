package app

import "embed"

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/style.css
var css string

//go:embed static/app.js
var js string
