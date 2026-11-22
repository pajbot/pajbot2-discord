package api

import "net/http"

func Init() {
	loginInit()
}

func Attach(handler *http.ServeMux) {
	loginAttach(handler)

	legalAttach(handler)
}
