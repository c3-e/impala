module goterm

go 1.16

require (
	app v0.0.0
	auth v0.0.0
	callback v0.0.0
	github.com/codegangsta/negroni v1.0.0
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/creack/pty v1.1.10
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/sessions v1.2.1 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	// Sean A. kr does not work, use creack
	// github.com/kr/pty v1.1.8 // indirect
	github.com/okta/okta-jwt-verifier-golang v1.1.1 // indirect
	github.com/quasoft/memstore v0.0.0-20191010062613-2bce066d2b0b // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	home v0.0.0
	login v0.0.0
	logout v0.0.0
	middlewares v0.0.0
	templates v0.0.0
	user v0.0.0
)

replace app => ./app

replace auth => ./auth

replace callback => ./routes/callback

replace home => ./routes/home

replace login => ./routes/login

replace logout => ./routes/logout

replace middlewares => ./routes/middlewares

replace user => ./routes/user

replace templates => ./routes/templates
