package implementation

import (
	"sync"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	urlpkg "github.com/tliron/kutil/url"
)

type DocumentState struct {
	Symbols     []protocol.DocumentSymbol
	Diagnostics []protocol.Diagnostic
}

var documentStates sync.Map // protocol.DocumentUri to DocumentState

func validateDocumentState(uri protocol.DocumentUri, notify glsp.NotifyFunc) *DocumentState {
	documentState, created := _getOrCreateDocumentState(uri)

	if created {
		go notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: documentState.Diagnostics,
		})
	}

	return documentState
}

func deleteDocumentState(uri protocol.DocumentUri) {
	documentStates.Delete(uri)
}

func _getOrCreateDocumentState(uri protocol.DocumentUri) (*DocumentState, bool) {
	if documentState, ok := documentStates.Load(uri); ok {
		return documentState.(*DocumentState), false
	} else {
		documentState := _createDocumentState(uri)
		if existing, loaded := documentStates.LoadOrStore(uri, documentState); loaded {
			return existing.(*DocumentState), false
		} else {
			return documentState, true
		}
	}
}

func _createDocumentState(uri protocol.DocumentUri) *DocumentState {
	var documentState DocumentState

	var err error
	// var url urlpkg.URL
	var content string
	// var context *parser.Context
	var node ast.Node

	urlContext := urlpkg.NewContext()
	defer urlContext.Release()

	path := uriToInternalPath(uri)
	content, _ = getDocument(uri)

	// if url, err = urlpkg.NewValidInternalURL(path, urlContext); err != nil {
	// 	log.Errorf("%s", err.Error())
	// 	documentState.Diagnostics = createDiagnostics(problems, content)
	// 	return &documentState
	// }

	node, err = jsonnet.SnippetToAST(path, content)

	if err != nil {
		log.Errorf("%s", err.Error())
	}

	opts := createSymbolsOptionalArgs{parentNode: &treeNode{node: node}}
	documentState.Symbols = createSymbols(node, content, uri, opts)

	return &documentState
}
