// internal/ui/details/model_test.go
package details_test

import (
	"strings"
	"testing"

	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/details"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/stretchr/testify/require"
)

func TestDetailsView(t *testing.T) {
	m := details.New(false)
	m2, _ := m.Update(msg.OpenDetails{Item: api.Item{
		Id: "m1", Name: "Dune", ProductionYear: 2021,
		Overview: "A noble family becomes embroiled in a war.",
	}})
	view := m2.(details.Model).View()
	require.True(t, strings.Contains(view, "Dune"), "title not in view: %q", view)
	require.True(t, strings.Contains(view, "2021"), "year not in view: %q", view)
}
