package shared

import "embed"

//go:embed templates/*
var TemplatesFS embed.FS

//go:embed templates/uploader.js
var StaticFS embed.FS
