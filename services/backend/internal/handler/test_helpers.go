package handler

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"mime/multipart"
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
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	return w
}

func performMultipartRequest(t *testing.T, r http.Handler, method, url, fileFieldName string, fileContent []byte, formFields map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if fileFieldName != "" && fileContent != nil {
		fw, err := w.CreateFormFile(fileFieldName, "upload.pdf")
		assert.NoError(t, err)
		_, err = fw.Write(fileContent)
		assert.NoError(t, err)
	}

	for key, val := range formFields {
		assert.NoError(t, w.WriteField(key, val))
	}
	assert.NoError(t, w.Close())

	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}
