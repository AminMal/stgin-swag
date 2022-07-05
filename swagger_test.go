package swag

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AminMal/stgin"
	"github.com/stretchr/testify/assert"
	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/swag"
)

type mockedSwag struct{}

func (s *mockedSwag) ReadDoc() string {
	return `{}`
}

func getSwaggerServer(swaggerInstanceName string) *stgin.Server {
	server := stgin.NewServer(":9000")
	route := stgin.GET(stgin.Prefix("/swagger"), WrapHandler(swaggerFiles.Handler, InstanceName(swaggerInstanceName)))
	server.AddRoutes(route)

	return server
}

func TestWrapHandler(t *testing.T) {
	server := getSwaggerServer("")

	w1 := performRequest(http.MethodGet, "/swagger/index.html", server)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, w1.Header()["Content-Type"][0], "text/html; charset=utf-8")
	w1BodyBytes, _ := ioutil.ReadAll(w1.Body)
	assert.Contains(t, string(w1BodyBytes), "doc.json")

	assert.Equal(t, http.StatusInternalServerError, performRequest(http.MethodGet, "/swagger/doc.json", server).Code)

	swag.Register(swag.Name, &mockedSwag{})

	w2 := performRequest(http.MethodGet, "/swagger/doc.json", server)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, w2.Header()["Content-Type"][0], "application/json; charset=utf-8")

	w3 := performRequest(http.MethodGet, "/swagger/favicon-16x16.png", server)
	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Equal(t, w3.Header()["Content-Type"][0], "image/png")

	w4 := performRequest(http.MethodGet, "/swagger/swagger-ui.css", server)
	assert.Equal(t, http.StatusOK, w4.Code)
	assert.Equal(t, w4.Header()["Content-Type"][0], "text/css; charset=utf-8")

	w5 := performRequest(http.MethodGet, "/swagger/swagger-ui-bundle.js", server)
	assert.Equal(t, http.StatusOK, w5.Code)
	assert.Equal(t, w5.Header()["Content-Type"][0], "application/javascript")

	assert.Equal(t, http.StatusNotFound, performRequest(http.MethodGet, "/swagger/notfound", server).Code)

}

func TestCustomWrapHandler(t *testing.T) {
	server := getSwaggerServer("custom")

	w1 := performRequest(http.MethodGet, "/swagger/index.html", server)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, w1.Header()["Content-Type"][0], "text/html; charset=utf-8")

	swag.Register("custom", &mockedSwag{})

	w2 := performRequest(http.MethodGet, "/swagger/doc.json", server)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, w2.Header()["Content-Type"][0], "application/json; charset=utf-8")

}

func performRequest(method, target string, server *stgin.Server) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	server.HttpHandler().ServeHTTP(w, r)

	return w
}

func TestURL(t *testing.T) {
	cfg := Config{}

	expected := "https://github.com/swaggo/http-swagger"
	configFunc := URL(expected)
	configFunc(&cfg)
	assert.Equal(t, expected, cfg.URL)
}

func TestDocExpansion(t *testing.T) {
	var cfg Config

	expected := "list"
	configFunc := DocExpansion(expected)
	configFunc(&cfg)
	assert.Equal(t, expected, cfg.DocExpansion)

	expected = "full"
	configFunc = DocExpansion(expected)
	configFunc(&cfg)
	assert.Equal(t, expected, cfg.DocExpansion)

	expected = "none"
	configFunc = DocExpansion(expected)
	configFunc(&cfg)
	assert.Equal(t, expected, cfg.DocExpansion)
}

func TestDeepLinking(t *testing.T) {
	var cfg Config
	assert.Equal(t, false, cfg.DeepLinking)

	configFunc := DeepLinking(true)
	configFunc(&cfg)
	assert.Equal(t, true, cfg.DeepLinking)

	configFunc = DeepLinking(false)
	configFunc(&cfg)
	assert.Equal(t, false, cfg.DeepLinking)

}

func TestDefaultModelsExpandDepth(t *testing.T) {
	var cfg Config

	assert.Equal(t, 0, cfg.DefaultModelsExpandDepth)

	expected := -1
	configFunc := DefaultModelsExpandDepth(expected)
	configFunc(&cfg)
	assert.Equal(t, expected, cfg.DefaultModelsExpandDepth)

	expected = 1
	configFunc = DefaultModelsExpandDepth(expected)
	configFunc(&cfg)
	assert.Equal(t, expected, cfg.DefaultModelsExpandDepth)
}

func TestInstanceName(t *testing.T) {
	var cfg Config

	assert.Equal(t, "", cfg.InstanceName)

	expected := swag.Name
	configFunc := InstanceName(expected)
	configFunc(&cfg)
	assert.Equal(t, expected, cfg.InstanceName)

	expected = "custom_name"
	configFunc = InstanceName(expected)
	configFunc(&cfg)
	assert.Equal(t, expected, cfg.InstanceName)
}

func TestPersistAuthorization(t *testing.T) {
	var cfg Config
	assert.Equal(t, false, cfg.PersistAuthorization)

	configFunc := PersistAuthorization(true)
	configFunc(&cfg)
	assert.Equal(t, true, cfg.PersistAuthorization)

	configFunc = PersistAuthorization(false)
	configFunc(&cfg)
	assert.Equal(t, false, cfg.PersistAuthorization)
}

func TestServedOnPrefix(t *testing.T) {
	server := stgin.NewServer(":9000")
	ServedOnPrefix("/swagger", server)

	w1 := performRequest(http.MethodGet, "/swagger/index.html", server)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, w1.Header()["Content-Type"][0], "text/html; charset=utf-8")
}
