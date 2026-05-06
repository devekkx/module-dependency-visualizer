package graph

import "errors"

var (
	ErrDuplicateNode = errors.New("graph: duplicate node")
	ErrUnknownNode   = errors.New("graph: unknown node")
	ErrUnknownEdge   = errors.New("graph: unknown edge")
	ErrSelfLoop      = errors.New("graph: self-loop not allowed")
)
