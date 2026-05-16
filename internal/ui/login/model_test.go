// internal/ui/login/model_test.go
package login_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/deglerj/fin/internal/ui/login"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/stretchr/testify/require"
)

func TestInitialView(t *testing.T) {
	m := login.New()
	require.NotEmpty(t, m.View())
}

func TestLoginErrorDisplayed(t *testing.T) {
	m := login.New()
	updated, _ := m.Update(msg.LoginError{Err: fmt.Errorf("bad credentials")})
	view := updated.(login.Model).View()
	require.True(t, strings.Contains(view, "bad credentials"), "error not shown in view: %q", view)
}
