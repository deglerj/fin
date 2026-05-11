// internal/ui/app/model_test.go
package app_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/deglerj/fin/internal/ui/app"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/stretchr/testify/require"
)

func TestStartsAtLogin(t *testing.T) {
	m := app.New(nil, nil, false)
	require.Equal(t, app.ScreenLogin, m.Screen())
}

func TestLoginSuccessTransition(t *testing.T) {
	m := app.New(nil, nil, false)
	m2, _ := m.Update(msg.LoginSuccess{ServerURL: "http://jf", UserID: "u1", AccessToken: "tok"})
	am := m2.(app.Model)
	require.Equal(t, app.ScreenBrowser, am.Screen())
}

func TestErrorDisplayed(t *testing.T) {
	m := app.New(nil, nil, false)
	m2, _ := m.Update(msg.AppError{Err: fmt.Errorf("network timeout")})
	view := m2.(app.Model).View()
	require.True(t, strings.Contains(view, "network timeout"), "error not in view: %q", view)
}
