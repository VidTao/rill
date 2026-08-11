package bratrax

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The quota only ever fires for a viewer, on the demo workspace, POSTing to one
// of the two AI entrypoints. Every other combination has to fall through
// untouched: this gate sits in front of the whole runtime API, so a match that
// is too broad would meter (and eventually 402) ordinary dashboard traffic.
func TestDemoAIQuota_Applies(t *testing.T) {
	q := &DemoAIQuota{clientSlug: "dummy", maxPrompts: 10}
	demoClient := &Client{ClientID: "uuid-demo", ClickhouseDB: "dummy"}
	realClient := &Client{ClientID: "uuid-real", ClickhouseDB: "vyne"}
	viewer := &User{ID: 1, Role: "viewer"}

	cases := []struct {
		name     string
		method   string
		pathTail string
		user     *User
		client   *Client
		want     bool
	}{
		{"streaming prompt", http.MethodPost, "/ai/complete/stream", viewer, demoClient, true},
		{"non-streaming prompt", http.MethodPost, "/ai/complete", viewer, demoClient, true},

		{"GET is not a prompt", http.MethodGet, "/ai/complete/stream", viewer, demoClient, false},
		{"other runtime path", http.MethodPost, "/queries/metrics-views/x", viewer, demoClient, false},
		{"conversations list", http.MethodPost, "/ai/conversations", viewer, demoClient, false},
		{"path prefix is not enough", http.MethodPost, "/ai/complete/stream/extra", viewer, demoClient, false},

		{"real client is never metered", http.MethodPost, "/ai/complete/stream", viewer, realClient, false},
		{"admin on demo is uncapped", http.MethodPost, "/ai/complete/stream", &User{ID: 2, Role: "admin"}, demoClient, false},
		{"super_admin on demo is uncapped", http.MethodPost, "/ai/complete/stream", &User{ID: 3, Role: "super_admin"}, demoClient, false},

		{"no user", http.MethodPost, "/ai/complete/stream", nil, demoClient, false},
		{"no client", http.MethodPost, "/ai/complete/stream", viewer, nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "/v1/instances/default"+tc.pathTail, nil)
			require.Equal(t, tc.want, q.applies(r, tc.user, tc.client, tc.pathTail))
		})
	}
}

// A nil quota is the "feature off" state (no platform key configured). It must
// meter nothing and block nothing, so demo users keep whatever behaviour they
// had before the feature existed.
func TestDemoAIQuota_NilIsInert(t *testing.T) {
	var q *DemoAIQuota
	r := httptest.NewRequest(http.MethodPost, "/v1/instances/default/ai/complete/stream", nil)
	require.False(t, q.applies(r, &User{ID: 1, Role: "viewer"}, &Client{ClickhouseDB: "dummy"}, "/ai/complete/stream"))
	require.True(t, q.admit(r.Context(), &User{ID: 1}, &Client{}, zap.NewNop()))
}

func TestNewDemoAIQuota_OffWithoutPlatformKey(t *testing.T) {
	store := &AIPromptStore{}
	require.Nil(t, NewDemoAIQuota(store, &Config{DemoClientSlug: "dummy"}))
	require.Nil(t, NewDemoAIQuota(nil, &Config{AnthropicAPIKey: "sk-ant-x"}))

	q := NewDemoAIQuota(store, &Config{
		AnthropicAPIKey:    "sk-ant-x",
		DemoClientSlug:     "dummy",
		DemoUsersModel:     "claude-sonnet-5",
		DemoUserMaxPrompts: 3,
	})
	require.NotNil(t, q)
	require.Equal(t, 3, q.maxPrompts)
	require.Equal(t, "claude-sonnet-5", q.model)
}
