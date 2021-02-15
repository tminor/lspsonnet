package implementation

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// protocol.InitializeFunc signature
func Initialize(context *glsp.Context, params *protocol.InitializeParams) (interface{}, error) {
	capabilities := Handler.CreateServerCapabilities()
	capabilities.TextDocumentSync = protocol.TextDocumentSyncKindIncremental
	capabilities.DefinitionProvider = true

	/*CompletionProvider: &protocol.CompletionOptions{
		TriggerCharacters: []string{" "},
	},
	SignatureHelpProvider: &protocol.SignatureHelpOptions{
		TriggerCharacters: []string{":"},
	},*/

	return &protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name: "jsonnet-lsp",
		},
	}, nil
}
