package implementation

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func symbolInRange(pos protocol.Position,symbolRange protocol.Range) bool {
	return pos.Line <= symbolRange.End.Line &&
		pos.Character <= symbolRange.End.Character &&
		symbolRange.Start.Line <= pos.Line &&
		symbolRange.Start.Character <= pos.Character
}

// TextDocumentDocumentSymbol implements protocol.TextDocumentDocumentSymbolFunc
func TextDocumentDocumentSymbol(context *glsp.Context, params *protocol.DocumentSymbolParams) (interface{}, error) {
	documentState := validateDocumentState(params.TextDocument.URI, context.Notify)
	return documentState.Symbols, nil
}

// TextDocumentDefinition implements protocol.TextDocumentDefinitionFunc
func TextDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (ret interface{}, err error) {
	var defLocation symbolLocation

	for location := range symbolDefinitions {
		if location.fileURI == params.TextDocument.URI && symbolInRange(params.TextDocumentPositionParams.Position, location.locRange) {
			defLocation = symbolDefinitions[location]
		}
	}
	definition := protocol.Location{
		URI: defLocation.fileURI,
		Range: defLocation.locRange,
	}

	return definition, nil
}
