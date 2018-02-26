package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// store interface that describes methods for reading and writing to a data store
type store interface {
	Get(string) ([]byte, error)
	Set(string, interface{}) error
	Delete(string)
	Append(string, string) error
	Pop(string) (string, error)
	MapGet(string, string) (string, error)
	MapSet(string, string, string) error
	MapDelete(string, string) error
	GetIndex(string) []string
}

// Response holds the reply from the server
type Response struct {
	Error string          `json:"error,omitempty"`
	Value json.RawMessage `json:"value"`
}

// Request represents an incoming command
type Request struct {
	Cmd   string      `json:"cmd"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// Server holds all methods that will listen and reply over an HTTP interface
type Server struct {
	s      store
	router *echo.Echo
}

// New creates a server instance. It takes an instance that fulfils interface store
func New(s store) *Server {
	var server Server
	server.s = s
	server.router = echo.New()
	return &server
}

// Serve listes for connections and starts the server
func (s *Server) Serve(port int) {
	s.router.Use(middleware.Logger())
	s.router.GET("/:key", s.Get)
	s.router.PUT("/:key", s.Set)
	s.router.DELETE("/:key", s.Delete)
	s.router.POST("/:key", s.Cmd)
	s.router.Logger.Fatal(s.router.Start(fmt.Sprintf(":%d", port)))
}

// Get the value of a key
func (s *Server) Get(c echo.Context) error {
	val, err := s.s.Get(c.Param("key"))
	var r Response
	if err != nil {
		r.Error = err.Error()
		return c.JSON(http.StatusBadRequest, r)
	}
	r.Value = val
	return c.JSON(http.StatusOK, r)
}

// Set a value on a key
func (s *Server) Set(c echo.Context) error {
	var request Request
	var r Response
	c.Bind(&request)
	err := s.s.Set(c.Param("key"), request.Value)
	if err != nil {
		r.Error = err.Error()
		return c.JSON(http.StatusBadRequest, r)
	}
	r.Value = []byte(fmt.Sprintf("\"%s\"", c.Param("key")))
	val, er := json.Marshal(r)
	fmt.Printf("%s, %v", val, er)
	return c.JSON(http.StatusOK, r)
}

// Delete removes a key
func (s *Server) Delete(c echo.Context) error {
	s.s.Delete(c.Param("key"))
	return c.NoContent(http.StatusNoContent)
}

// Cmd runs a command against the key
func (s *Server) Cmd(c echo.Context) error {
	var request Request
	var r Response
	var err error
	var rcode = http.StatusOK
	var value string
	var res string
	c.Bind(&request)
	key := c.Param("key")
	if request.Cmd == "append" || request.Cmd == "mapset" {
		var ok bool
		value, ok = request.Value.(string)
		if !ok {
			r.Error = "value must be a string"
			return c.JSON(http.StatusBadRequest, r)
		}
	}
	switch request.Cmd {
	case "append":
		err = s.s.Append(key, value)
		break
	case "pop":
		res, err = s.s.Pop(key)
		if err == nil {
			r.Value, err = json.Marshal(res)
		}
		break
	case "mapget":
		res, err = s.s.MapGet(key, request.Key)
		if err == nil {
			r.Value, err = json.Marshal(res)
		}
		break
	case "mapset":
		err = s.s.MapSet(key, request.Key, value)
		break
	case "mapdelete":
		err = s.s.MapDelete(key, request.Key)
		break
	case "index":
		index := s.s.GetIndex(request.Key)
		r.Value, err = json.Marshal(index)
		break
	default:
		err = fmt.Errorf("no command specified")
	}
	if err != nil {
		r.Error = err.Error()
		rcode = http.StatusBadRequest
	}
	return c.JSON(rcode, r)
}
