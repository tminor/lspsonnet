package implementation

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// documentStore holds opened documents.
type documentStore struct {
	documents map[string]*document
}

// document represents a parsed Jsonnet file.
type document struct {
	URI                     protocol.DocumentUri
	Path                    string
	Content                 string
}

type node struct {
	uniqueIdentifier string
	parent           *protocol.DocumentSymbol
	symbol           *protocol.DocumentSymbol
	children         []*protocol.DocumentSymbol
}
