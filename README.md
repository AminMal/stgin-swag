# stgin-swagger

stgin add-on library to generate RESTful API documentation with Swagger 2.0.

## Usage

### Start using it
1. Add comments to your API source code, [See Declarative Comments Format](https://github.com/swaggo/swag#declarative-comments-format).
2. Download [Swag](https://github.com/swaggo/swag) for Go by using:
```sh
$ go get -d github.com/swaggo/swag/cmd/swag

# 1.16 or newer
$ go install github.com/swaggo/swag/cmd/swag@latest
```
3. Run the [Swag](https://github.com/swaggo/swag) in your Go project root folder which contains `main.go` file, [Swag](https://github.com/swaggo/swag) will parse comments and generate required files(`docs` folder and `docs/doc.go`).
```sh
$ swag init
```
4. Download [stgin-swag](https://github.com/AminMal/stgin-swag) by using:
```sh
$ go get -u github.com/AminMal/stgin-swag
```
And import following in your code:
```go
import swag "github.com/AminMal/stgin-swag"
```
After (not necessarily) adding routes to the server, you can use this utility function called `ServedOnPrefix`, to add swagger to your server.
```go
server.AddRoutes(...)
swag.ServedOnPrefix("/swagger", server, myCustomOptions...)
```
5. Run it, and browser to http://localhost:1323/swagger/index.html, you can see Swagger 2.0 Api documents.

![swagger_index.html](https://user-images.githubusercontent.com/8943871/36250587-40834072-1279-11e8-8bb7-02a2e2fdd7a7.png)

