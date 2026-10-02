package service

import (
	"bytes"
	"html/template"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationTemplates_RenderWelcome(t *testing.T) {
	// Locate template file relative to this test
	tplPath := filepath.Join("..", "templates", "welcome.html")
	tpl, err := template.ParseFiles(tplPath)
	require.NoError(t, err)

	data := map[string]any{
		"first_name": "Ahmet",
		"role":       "student",
		"login_url":  "https://mydreamcampus.edu.tr/auth/login",
	}

	var buf bytes.Buffer
	err = tpl.ExecuteTemplate(&buf, "welcome.html", data)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Hoş geldin Ahmet!")
	assert.Contains(t, output, "Rol: student")
	assert.Contains(t, output, `href="https://mydreamcampus.edu.tr/auth/login"`)
}

func TestNotificationTemplates_RenderPasswordReset(t *testing.T) {
	tplPath := filepath.Join("..", "templates", "password_reset.html")
	tpl, err := template.ParseFiles(tplPath)
	require.NoError(t, err)

	token := "secure-token-12345"
	resetURL := "https://mydreamcampus.edu.tr/auth/reset-password?token=" + url.QueryEscape(token)
	expiresAt := "15 dakika"

	data := map[string]any{
		"expires_at": expiresAt,
		"reset_url":  resetURL,
	}

	var buf bytes.Buffer
	err = tpl.ExecuteTemplate(&buf, "password_reset.html", data)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, resetURL)
	assert.Contains(t, output, "15 dakika")
}
