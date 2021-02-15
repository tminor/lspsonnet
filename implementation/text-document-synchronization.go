package implementation

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TextDocumentDidOpen implements protocol.TextDocumentDidOpenFunc
func TextDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	setDocument(params.TextDocument.URI, params.TextDocument.Text)
	go validateDocumentState(params.TextDocument.URI, context.Notify)
	return nil
}

// TextDocumentDidChange implements protocol.TextDocumentDidChangeFunc
func TextDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	if content, ok := getDocument(params.TextDocument.URI); ok {
		for _, change := range params.ContentChanges {
			if change_, ok := change.(protocol.TextDocumentContentChangeEvent); ok {
				startIndex, endIndex := rangeToIndex(content, &change_.Range)
				content = content[:startIndex] + change_.Text + content[endIndex:]
			} else if change_, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
				content = change_.Text
			}
		}
		setDocument(params.TextDocument.URI, content)
		go validateDocumentState(params.TextDocument.URI, context.Notify)
	}
	return nil
}

// TextDocumentDidSave implements protocol.TextDocumentDidSaveFunc
func TextDocumentDidSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	return nil
}

// TextDocumentDidClose implements protocol.TextDocumentDidCloseFunc
func TextDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	deleteDocumentState(params.TextDocument.URI)
	deleteDocument(params.TextDocument.URI)

	go context.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: []protocol.Diagnostic{},
	})

	return nil
}
