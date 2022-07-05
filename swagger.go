package swag

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"golang.org/x/net/webdav"

	"github.com/AminMal/slogger"
	"github.com/AminMal/slogger/colored"
	"github.com/AminMal/stgin"
	"github.com/swaggo/swag"
)

var swaggerLogger = slogger.NewConsoleLogger("swagger")

type swaggerConfig struct {
	URL                      string
	DocExpansion             string
	Title                    string
	Oauth2RedirectURL        template.JS
	DefaultModelsExpandDepth int
	DeepLinking              bool
	PersistAuthorization     bool
	Oauth2DefaultClientID    string
}

// Config stores ginSwagger configuration variables.
type Config struct {
	// The url pointing to API definition (normally swagger.json or swagger.yaml). Default is `doc.json`.
	URL                      string
	DocExpansion             string
	InstanceName             string
	Title                    string
	DefaultModelsExpandDepth int
	DeepLinking              bool
	PersistAuthorization     bool
	Oauth2DefaultClientID    string
}

func (config Config) toSwaggerConfig() swaggerConfig {
	return swaggerConfig{
		URL:                      config.URL,
		DeepLinking:              config.DeepLinking,
		DocExpansion:             config.DocExpansion,
		DefaultModelsExpandDepth: config.DefaultModelsExpandDepth,
		Oauth2RedirectURL: "`${window.location.protocol}//${window.location.host}$" +
			"{window.location.pathname.split('/').slice(0, window.location.pathname.split('/').length - 1).join('/')}" +
			"/oauth2-redirect.html`",
		Title:                 config.Title,
		PersistAuthorization:  config.PersistAuthorization,
		Oauth2DefaultClientID: config.Oauth2DefaultClientID,
	}
}

// URL presents the url pointing to API definition (normally swagger.json or swagger.yaml).
func URL(url string) func(*Config) {
	return func(c *Config) {
		c.URL = url
	}
}

// DocExpansion list, full, none.
func DocExpansion(docExpansion string) func(*Config) {
	return func(c *Config) {
		c.DocExpansion = docExpansion
	}
}

// DeepLinking set the swagger deep linking configuration.
func DeepLinking(deepLinking bool) func(*Config) {
	return func(c *Config) {
		c.DeepLinking = deepLinking
	}
}

// DefaultModelsExpandDepth set the default expansion depth for models
// (set to -1 completely hide the models).
func DefaultModelsExpandDepth(depth int) func(*Config) {
	return func(c *Config) {
		c.DefaultModelsExpandDepth = depth
	}
}

// InstanceName set the instance name that was used to generate the swagger documents
// Defaults to swag.Name ("swagger").
func InstanceName(name string) func(*Config) {
	return func(c *Config) {
		c.InstanceName = name
	}
}

// PersistAuthorization Persist authorization information over browser close/refresh.
// Defaults to false.
func PersistAuthorization(persistAuthorization bool) func(*Config) {
	return func(c *Config) {
		c.PersistAuthorization = persistAuthorization
	}
}

// Oauth2DefaultClientID set the default client ID used for OAuth2
func Oauth2DefaultClientID(oauth2DefaultClientID string) func(*Config) {
	return func(c *Config) {
		c.Oauth2DefaultClientID = oauth2DefaultClientID
	}
}

// WrapHandler wraps `http.Handler` into `gin.HandlerFunc`.
func WrapHandler(handler *webdav.Handler, options ...func(*Config)) stgin.API {
	var config = Config{
		URL:                      "doc.json",
		DocExpansion:             "list",
		InstanceName:             swag.Name,
		Title:                    "Swagger UI",
		DefaultModelsExpandDepth: 1,
		DeepLinking:              true,
		PersistAuthorization:     false,
		Oauth2DefaultClientID:    "",
	}

	for _, c := range options {
		c(&config)
	}

	return CustomWrapHandler(&config, handler)
}


// costum stgin.ResponseEntity

type responseEntity struct {
	contentType 	string
	bytes 			[]byte
	err 			error
}

func (r responseEntity) ContentType() string { return r.contentType }
func (r responseEntity) Bytes() ([]byte, error) {
	return r.bytes, r.err
}

// ==== Aggregator sink ====

type sink struct {
	bytes []byte
}

func (s *sink) Write(p []byte) (n int, err error) {
	s.bytes = append(s.bytes, p...)
	return len(p), nil
}

// ========================


// CustomWrapHandler wraps `http.Handler` into `gin.HandlerFunc`.
func CustomWrapHandler(config *Config, handler *webdav.Handler) stgin.API {
	var once sync.Once

	if config.InstanceName == "" {
		config.InstanceName = swag.Name
	}

	if config.Title == "" {
		config.Title = "Swagger UI"
	}

	// create a template with name
	index, _ := template.New("swagger_index.html").Parse(swaggerIndexTpl)

	var matcher = regexp.MustCompile(`(.*)(index\.html|doc\.json|favicon-16x16\.png|favicon-32x32\.png|/oauth2-redirect\.html|swagger-ui\.css|swagger-ui\.css\.map|swagger-ui\.js|swagger-ui\.js\.map|swagger-ui-bundle\.js|swagger-ui-bundle\.js\.map|swagger-ui-standalone-preset\.js|swagger-ui-standalone-preset\.js\.map)[?|.]*`)

	return func(request stgin.RequestContext) stgin.Status {
		if request.Method != http.MethodGet {
			return stgin.MethodNotAllowed(stgin.Empty())
		}

		matches := matcher.FindStringSubmatch(request.Url)

		if len(matches) != 3 {
			return stgin.NotFound(stgin.Text(http.StatusText(http.StatusNotFound)))
		}

		path := matches[2]
		once.Do(func() {
			handler.Prefix = matches[1]
		})

		contentType := ""
		switch filepath.Ext(path) {
		case ".html":
			contentType = "text/html; charset=utf-8"
		case ".css":
			contentType = "text/css; charset=utf-8"
		case ".js":
			contentType = "application/javascript"
		case ".png":
			contentType = "image/png"
		case ".json":
			contentType = "application/json; charset=utf-8"
		}

		aggregator := sink{[]byte{}}
		switch path {
		case "index.html":
			err := index.Execute(&aggregator, config.toSwaggerConfig())
			if err != nil {
				swaggerLogger.Colored(colored.RED).Err(err.Error())
				return stgin.InternalServerError(stgin.Text("internal server error"))
			} else {
				entity := responseEntity{
					contentType: contentType,
					bytes: aggregator.bytes,
					err: nil,
				}
				return stgin.Ok(entity)
			}
		case "doc.json":
			doc, err := swag.ReadDoc(config.InstanceName)
			docMap := make(map[string]any)
			if err != nil {
				swaggerLogger.Colored(colored.RED).Err(err.Error())
				return stgin.InternalServerError(stgin.Text("internal server error"))
			}
			err = json.Unmarshal([]byte(doc), &docMap)
			if err != nil {
				swaggerLogger.Colored(colored.RED).Err(err.Error())
				return stgin.InternalServerError(stgin.Text("internal server error"))
			}
			return stgin.Ok(stgin.Json(&docMap))
		}
		return stgin.Ok(stgin.Empty())

	}
}

// DisablingWrapHandler turn handler off
// if specified environment variable passed.
func DisablingWrapHandler(handler *webdav.Handler, envName string) stgin.API {
	if os.Getenv(envName) != "" {
		return func(stgin.RequestContext) stgin.Status {
			return stgin.NotFound(stgin.Empty())
		}
	}

	return WrapHandler(handler)
}

// DisablingCustomWrapHandler turn handler off
// if specified environment variable passed.
func DisablingCustomWrapHandler(config *Config, handler *webdav.Handler, envName string) stgin.API {
	if os.Getenv(envName) != "" {
		return func(stgin.RequestContext) stgin.Status {
			return stgin.NotFound(stgin.Empty())
		}
	}
	return CustomWrapHandler(config, handler)
}

const swaggerIndexTpl = `<!-- HTML for static distribution bundle build -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{.Title}}</title>
  <link href="https://fonts.googleapis.com/css?family=Open+Sans:400,700|Source+Code+Pro:300,600|Titillium+Web:400,600,700" rel="stylesheet">
  <link rel="stylesheet" type="text/css" href="./swagger-ui.css" >
  <link rel="icon" type="image/png" href="./favicon-32x32.png" sizes="32x32" />
  <link rel="icon" type="image/png" href="./favicon-16x16.png" sizes="16x16" />
  <style>
    html
    {
        box-sizing: border-box;
        overflow: -moz-scrollbars-vertical;
        overflow-y: scroll;
    }
    *,
    *:before,
    *:after
    {
        box-sizing: inherit;
    }
    body {
      margin:0;
      background: #fafafa;
    }
  </style>
</head>
<body>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" style="position:absolute;width:0;height:0">
  <defs>
    <symbol viewBox="0 0 20 20" id="unlocked">
          <path d="M15.8 8H14V5.6C14 2.703 12.665 1 10 1 7.334 1 6 2.703 6 5.6V6h2v-.801C8 3.754 8.797 3 10 3c1.203 0 2 .754 2 2.199V8H4c-.553 0-1 .646-1 1.199V17c0 .549.428 1.139.951 1.307l1.197.387C5.672 18.861 6.55 19 7.1 19h5.8c.549 0 1.428-.139 1.951-.307l1.196-.387c.524-.167.953-.757.953-1.306V9.199C17 8.646 16.352 8 15.8 8z"></path>
    </symbol>
    <symbol viewBox="0 0 20 20" id="locked">
      <path d="M15.8 8H14V5.6C14 2.703 12.665 1 10 1 7.334 1 6 2.703 6 5.6V8H4c-.553 0-1 .646-1 1.199V17c0 .549.428 1.139.951 1.307l1.197.387C5.672 18.861 6.55 19 7.1 19h5.8c.549 0 1.428-.139 1.951-.307l1.196-.387c.524-.167.953-.757.953-1.306V9.199C17 8.646 16.352 8 15.8 8zM12 8H8V5.199C8 3.754 8.797 3 10 3c1.203 0 2 .754 2 2.199V8z"/>
    </symbol>
    <symbol viewBox="0 0 20 20" id="close">
      <path d="M14.348 14.849c-.469.469-1.229.469-1.697 0L10 11.819l-2.651 3.029c-.469.469-1.229.469-1.697 0-.469-.469-.469-1.229 0-1.697l2.758-3.15-2.759-3.152c-.469-.469-.469-1.228 0-1.697.469-.469 1.228-.469 1.697 0L10 8.183l2.651-3.031c.469-.469 1.228-.469 1.697 0 .469.469.469 1.229 0 1.697l-2.758 3.152 2.758 3.15c.469.469.469 1.229 0 1.698z"/>
    </symbol>
    <symbol viewBox="0 0 20 20" id="large-arrow">
      <path d="M13.25 10L6.109 2.58c-.268-.27-.268-.707 0-.979.268-.27.701-.27.969 0l7.83 7.908c.268.271.268.709 0 .979l-7.83 7.908c-.268.271-.701.27-.969 0-.268-.269-.268-.707 0-.979L13.25 10z"/>
    </symbol>
    <symbol viewBox="0 0 20 20" id="large-arrow-down">
      <path d="M17.418 6.109c.272-.268.709-.268.979 0s.271.701 0 .969l-7.908 7.83c-.27.268-.707.268-.979 0l-7.908-7.83c-.27-.268-.27-.701 0-.969.271-.268.709-.268.979 0L10 13.25l7.418-7.141z"/>
    </symbol>
    <symbol viewBox="0 0 24 24" id="jump-to">
      <path d="M19 7v4H5.83l3.58-3.59L8 6l-6 6 6 6 1.41-1.41L5.83 13H21V7z"/>
    </symbol>
    <symbol viewBox="0 0 24 24" id="expand">
      <path d="M10 18h4v-2h-4v2zM3 6v2h18V6H3zm3 7h12v-2H6v2z"/>
    </symbol>
  </defs>
</svg>
<div id="swagger-ui"></div>
<script src="./swagger-ui-bundle.js"> </script>
<script src="./swagger-ui-standalone-preset.js"> </script>
<script>
window.onload = function() {
  // Build a system
  const ui = SwaggerUIBundle({
    url: "{{.URL}}",
    dom_id: '#swagger-ui',
    validatorUrl: null,
    oauth2RedirectUrl: {{.Oauth2RedirectURL}},
    persistAuthorization: {{.PersistAuthorization}},
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
	layout: "StandaloneLayout",
    docExpansion: "{{.DocExpansion}}",
	deepLinking: {{.DeepLinking}},
	defaultModelsExpandDepth: {{.DefaultModelsExpandDepth}}
  })
  const defaultClientId = "{{.Oauth2DefaultClientID}}";
  if (defaultClientId) {
    ui.initOAuth({
      clientId: defaultClientId
    })
  }
  window.ui = ui
}
</script>
</body>
</html>
`
