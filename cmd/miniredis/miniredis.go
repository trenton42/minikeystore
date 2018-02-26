package main

import (
	"github.com/trenton42/miniredis/internal/server"
	"github.com/trenton42/miniredis/internal/storage"
)

func main() {
	v := storage.New()
	s := server.New(v)
	s.Serve(8787)
}
