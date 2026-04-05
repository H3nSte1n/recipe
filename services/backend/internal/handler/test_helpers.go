package handler

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mustJson(t *testing.T, v any) []byte {
	t.Helper()

	responseJSON, err := json.Marshal(v)
	assert.NoError(t, err)

	return responseJSON
}

func performRequest(r http.Handler, method, url string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, bytes.NewReader(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	return w
}
