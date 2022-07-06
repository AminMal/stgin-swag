package swag

import (
	"encoding/json"
	swaggerFiles "github.com/swaggo/files"
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

// Config stores stginSwagger configuration variables.
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

// WrapHandler wraps `http.Handler` into `stgin.API`.
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


// custom stgin.ResponseEntity

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

type responseAggregator struct {
	statusCode 	int
	entity 		[]byte
	headers     http.Header
}

func (r *responseAggregator) Header() http.Header {
	return r.headers
}

func (r *responseAggregator) Write(bytes []byte) (int, error) {
	r.entity = append(r.entity, bytes...)
	return len(bytes), nil
}

func (r *responseAggregator) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}


// CustomWrapHandler wraps `http.Handler` into `stgin.API`.
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
			btes, err := json.Marshal(&docMap)
			entity := responseEntity{
				contentType: contentType,
				bytes:       btes,
				err:         err,
			}
			return stgin.Ok(entity)
		default:
			response := responseAggregator{
				statusCode: 0,
				entity:     []byte{},
				headers:    http.Header{},
			}
			handler.ServeHTTP(&response, request.Underlying)
			if response.statusCode <= 0 { response.statusCode = 200 }
			return stgin.CreateResponse(response.statusCode, responseEntity{
				contentType: contentType,
				bytes:       response.entity,
			})
		}
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

func ServedOnPrefix(prefix string, server *stgin.Server, options ...func(*Config)) {
	server.AddRoutes(
		stgin.GET(stgin.Prefix(prefix), WrapHandler(swaggerFiles.Handler, options...)),
	)
}
