package redka

import (
	"fmt"
	"sync"

	"github.com/nalgeon/redka"
	"github.com/nalgeon/redka/redsrv"
	_ "modernc.org/sqlite"
)

type Server struct {
	db   *redka.DB
	resp *redsrv.Server
	once sync.Once
}

func NewServer(dbPath, respAddr string) (*Server, error) {
	db, err := redka.Open(dbPath, &redka.Options{DriverName: "sqlite"})
	if err != nil {
		return nil, fmt.Errorf("open redka db: %w", err)
	}

	return &Server{
		db:   db,
		resp: redsrv.New("tcp", respAddr, db),
	}, nil
}

func (s *Server) Start() error {
	ready := make(chan error, 1)
	go func() {
		if err := s.resp.Start(ready); err != nil {
			ready <- err
		}
	}()

	if err := <-ready; err != nil {
		return err
	}
	return nil
}

func (s *Server) Close() error {
	var err error
	s.once.Do(func() {
		err = s.resp.Stop()
	})
	return err
}

func (s *Server) DB() *redka.DB {
	return s.db
}
