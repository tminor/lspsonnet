package implementation

import (
	"fmt"
	"path/filepath"
	//"path/filepath"
	"regexp"
	"strings"

	"github.com/google/go-jsonnet/ast"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type symbolLocation struct {
	fileURI  protocol.DocumentUri
	position protocol.Position
	locRange protocol.Range
}

// symbolDefinitions provides definition locations for symbols
var symbolDefinitions map[symbolLocation]symbolLocation
// symbolReferences provides reference locations for symbols
var symbolReferences map[symbolLocation][]symbolLocation

// treeNode represents a symbol in an AST
type treeNode struct {
	node           interface{}
	documentSymbol *protocol.DocumentSymbol
	parentNode     *treeNode
	childNodes     []*treeNode
}

func (ancestor *treeNode) findObjectIndex(index string) *symbolLocation {
	parent := ancestor

	for _, ok := parent.node.(*ast.DesugaredObject); !ok; _, ok = parent.node.(*ast.DesugaredObject) {
		parent = parent.parentNode
		if parent == nil {
			break
		}
	}

	parentObj, ok := parent.node.(*ast.DesugaredObject)

	if !ok {
		return nil
	}

	var foundField *ast.DesugaredObjectField

	for _, field := range parentObj.Fields {
		fieldName, ok := field.Name.(*ast.LiteralString)
		if ok {
			name := fieldName.Value
			if name == index {
				foundField = &field
				break
			}
		}
	}

	path, _ := getFileURI(string(foundField.LocRange.File.DiagnosticFileName))

	return &symbolLocation{
		fileURI: path,
		position: protocol.Position{
			Line:      protocol.UInteger(foundField.LocRange.Begin.Line - 1),
			Character: protocol.UInteger(foundField.LocRange.Begin.Column - 1),
		},
		locRange: protocol.Range{
			Start: protocol.Position{
				Line:      protocol.UInteger(foundField.LocRange.Begin.Line - 1),
				Character: protocol.UInteger(foundField.LocRange.Begin.Column - 1),
			},
			End:   protocol.Position{
				Line:      protocol.UInteger(foundField.LocRange.End.Line - 1),
				Character: protocol.UInteger(foundField.LocRange.End.Column - 1),
			},
		},
	}
}

func (ancestor *treeNode) findObjectIndexLocRange(indexName string) *symbolLocation {
	index, ok := ancestor.node.(*ast.Index)

	if !ok {
		return nil
	}

	lineNum := index.LocRange.Begin.Line - 1
	lineStr := index.LocRange.File.Lines[lineNum]

	begCol := strings.Index(lineStr, indexName)
	endCol := (begCol + len(indexName)) - 1

	if begCol > 0 {
		path, err := getFileURI(string(index.LocRange.File.DiagnosticFileName))

		if err != nil {
			return nil
		}

		return &symbolLocation{
			fileURI: path,
			position: protocol.Position{
				Line: protocol.UInteger(lineNum),
				Character: protocol.UInteger(begCol),
			},
			locRange: protocol.Range{
				Start: protocol.Position{
					Line: protocol.UInteger(lineNum),
					Character: protocol.UInteger(begCol),
				},
				End: protocol.Position{
					Line: protocol.UInteger(lineNum),
					Character: protocol.UInteger(endCol),
				},
			},
		}
	} else {
		return nil
	}
}

// createSymbolsOptionalArgs is used for optional arguments passed to createSymbols
type createSymbolsOptionalArgs struct {
	parentNode *treeNode
}

func getFileURI(path string) (filePath string, err error) {
	filePath, err = filepath.Abs(path)

	pathRegex := regexp.MustCompile("(?:[^:]+:)+(.+)")
	match := pathRegex.FindStringSubmatch(filePath)

	log.Debugf("%s", filePath)

	if len(match) > 0 {
		filePath = match[1]
	}

	filePath = fmt.Sprintf("file://%s", filePath)

	log.Debugf("%s", filePath)

	if err != nil {
		return "", err
	} else {
		return filePath, err
	}
}

// createSymbols creates protocol.DocumentSymbols and populates
// definition and reference maps.
func createSymbols(node interface{}, content string, uri protocol.DocumentUri, optionalArgs createSymbolsOptionalArgs) []protocol.DocumentSymbol {
	var symbols []protocol.DocumentSymbol

	if symbolDefinitions == nil {
		symbolDefinitions = make(map[symbolLocation]symbolLocation)
	}
	if symbolReferences == nil {
		symbolReferences = make(map[symbolLocation][]symbolLocation)
	}
	thisSymbol := treeNode{
		parentNode: optionalArgs.parentNode,
	}
	if optionalArgs.parentNode.childNodes == nil {
		optionalArgs.parentNode.childNodes = make([]*treeNode, 0)
	}
	optionalArgs.parentNode.childNodes = append(optionalArgs.parentNode.childNodes, &thisSymbol)
	optionalArgs.parentNode = &thisSymbol

	switch t := node.(type) {
	default:
	case *ast.Apply:
		thisSymbol.node = t
		symbols = append(symbols, createSymbols(t.Target, content, uri, optionalArgs)...)
		// The child of this node may be an *ast.Index which
		// should contain the name of the function being
		// applied.
		for _, child := range thisSymbol.childNodes {
			index, ok := child.node.(*ast.Index)
			if ok {
				// Found an index, now we use it to
				// find its definition in the parent
				// object.
				funcName, ok := index.Index.(*ast.LiteralString)
				if ok {
					name := string(funcName.Value)
					funcDef := thisSymbol.findObjectIndex(name)
					indexLoc := child.findObjectIndexLocRange(name)
					indexLoc.fileURI, _ = getFileURI(string(index.LocRange.File.DiagnosticFileName))
					if funcDef != nil {
						log.Debugf("%s; %s", indexLoc, funcDef)
						symbolDefinitions[*indexLoc] = *funcDef
					}
				}
			}
		}
	case ast.Arguments:
	case *ast.ApplyBrace:
	case *ast.Array:
	case *ast.ArrayComp:
	case *ast.Assert:
	case *ast.Binary:
	case ast.CommaSeparatedExpr:
	case *ast.Conditional:
	case *ast.Dollar:
	case *ast.DesugaredObject:
		thisSymbol.node = t
		start := protocol.Position{
			Line:      protocol.UInteger(t.NodeBase.LocRange.Begin.Line - 1),
			Character: protocol.UInteger(t.NodeBase.LocRange.Begin.Column - 1),
		}
		end := protocol.Position{
			Line:      protocol.UInteger(t.NodeBase.LocRange.End.Line - 1),
			Character: protocol.UInteger(t.NodeBase.LocRange.End.Column - 1),
		}
		symbolRange := protocol.Range{
			Start: start,
			End:   end,
		}
		obj := protocol.DocumentSymbol{
			Name:  fmt.Sprintf("anonymous object <%v>", &node),
			Kind:  protocol.SymbolKindObject,
			Range: symbolRange,
		}
		if len(t.Fields) > 0 {
			for _, field := range t.Fields {
				obj.Children = append(obj.Children, createSymbols(field, content, uri, optionalArgs)...)
			}
		}
		if len(t.Locals) > 0 {
			for _, local := range t.Locals {
				obj.Children = append(obj.Children, createSymbols(local, content, uri, optionalArgs)...)
			}
		}
		symbols = append(symbols, obj)
	case *ast.DesugaredObjectField:
	case ast.DesugaredObjectField:
		thisSymbol.node = t
		if t.Name.(*ast.LiteralString) != nil {
			start := protocol.Position{
				Line:      protocol.UInteger(t.LocRange.Begin.Line - 1),
				Character: protocol.UInteger(t.LocRange.Begin.Column - 1),
			}
			end := protocol.Position{
				Line:      protocol.UInteger(t.LocRange.End.Line - 1),
				Character: protocol.UInteger(t.LocRange.End.Column - 1),
			}
			field := protocol.DocumentSymbol{
				Name:  string(t.Name.(*ast.LiteralString).Value),
				Kind:  protocol.SymbolKindField,
				Range: protocol.Range{
					Start: start,
					End:   end,
				},
			}
			symbols = append(symbols, field)
			field.Children = append(field.Children, createSymbols(t.Body, content, uri, optionalArgs)...)
		}
	case *ast.Error:
	case *ast.Function:
		thisSymbol.node = t
		start := protocol.Position{
			Line:      protocol.UInteger(t.NodeBase.LocRange.Begin.Line - 1),
			Character: protocol.UInteger(t.NodeBase.LocRange.Begin.Column - 1),
		}
		end := protocol.Position{
			Line:      protocol.UInteger(t.NodeBase.LocRange.End.Line - 1),
			Character: protocol.UInteger(t.NodeBase.LocRange.End.Column - 1),
		}
		function := protocol.DocumentSymbol{
			Name:  fmt.Sprintf("%v", &node),
			Kind:  protocol.SymbolKindObject,
			Range: protocol.Range{
				Start: start,
				End:   end,
			},
		}
		symbols = append(symbols, function)
		function.Children = append(function.Children, createSymbols(t.Body, content, uri, optionalArgs)...)
	case ast.IfSpec:
	case *ast.Import:
	case *ast.ImportStr:
	case *ast.Index:
		thisSymbol.node = t
		symbols = append(symbols, createSymbols(t.Target, content, uri, optionalArgs)...)
	case *ast.InSuper:
	case *ast.Local:
		// for _, bind := range t.Binds {
		// 	start := protocol.Position{
		// 		Line:      protocol.UInteger(bind.LocRange.Begin.Line - 1),
		// 		Character: protocol.UInteger(bind.LocRange.Begin.Column - 1),
		// 	}
		// 	end := protocol.Position{
		// 		Line: protocol.UInteger(bind.LocRange.End.Line - 1),
		// 		Character: protocol.UInteger(bind.LocRange.End.Column - 1),
		// 	}
		// 	symbols = append(symbols, protocol.SymbolInformation{
		// 		Name: string(bind.Variable),
		// 		Kind: protocol.SymbolKindVariable,
		// 		Location: protocol.Location{
		// 			URI: uri,
		// 			Range: protocol.Range{
		// 				Start: start,
		// 				End: end,
		// 			},
		// 		},
		// 	})
		// 	symbols = append(symbols, createSymbols(bind.Body, content, uri)...)
		// }
	case ast.LocalBind:
	case *ast.Object:
	case *ast.ObjectComp:
	case ast.ObjectField:
	case *ast.LiteralBoolean:
	case *ast.LiteralString:
	case *ast.LiteralNumber:
	case *ast.Parens:
	case *ast.Self:
		thisSymbol.node = t

		var parentObj *ast.DesugaredObject
		var location symbolLocation
		var parentLocation symbolLocation

		ancestor := *optionalArgs.parentNode

		// Traverse ancestors until we find an object (self
		// can only be an object):
		for _, ok := ancestor.node.(*ast.DesugaredObject); !ok; _, ok = ancestor.node.(*ast.DesugaredObject) {
			ancestor = *ancestor.parentNode
		}

		parentObj, _ = ancestor.node.(*ast.DesugaredObject)

		symbolStart := protocol.Position{
			Line:      protocol.UInteger(t.LocRange.Begin.Line - 1),
			Character: protocol.UInteger(t.LocRange.Begin.Column - 1),
		}
		symbolEnd := protocol.Position{
			Line:      protocol.UInteger(t.LocRange.End.Line - 1),
			Character: protocol.UInteger(t.LocRange.End.Column - 1),
		}

		location.position = symbolStart
		location.locRange = protocol.Range{
			Start: symbolStart,
			End:   symbolEnd,
		}

		uriAbsPath, err := getFileURI(string(t.LocRange.File.DiagnosticFileName))

		if err != nil {
			return nil
		}

		location.fileURI = uriAbsPath

		parentStart := protocol.Position{
			Line:      protocol.UInteger(parentObj.LocRange.Begin.Line - 1),
			Character: protocol.UInteger(parentObj.LocRange.Begin.Column - 1),
		}
		parentEnd := protocol.Position{
			Line:      protocol.UInteger(parentObj.LocRange.End.Line - 1),
			Character: protocol.UInteger(parentObj.LocRange.End.Column - 1),
		}

		parentLocation.position = parentStart
		parentLocation.locRange = protocol.Range{
			Start: parentStart,
			End:   parentEnd,
		}

		parentFilePath, err := getFileURI(string(parentObj.LocRange.File.DiagnosticFileName))

		if err != nil {
			return nil
		}

		parentLocation.fileURI = parentFilePath

		if symbolReferences[parentLocation] == nil {
			symbolReferences[parentLocation] = make([]symbolLocation, 0)
		}

		symbolReferences[parentLocation] = append(symbolReferences[parentLocation], location)
		symbolDefinitions[location] = parentLocation
	case *ast.Slice:
	case *ast.Unary:
	case *ast.Var:
		thisSymbol.node = t
		start := protocol.Position{
			Line:      protocol.UInteger(t.LocRange.Begin.Line - 1),
			Character: protocol.UInteger(t.LocRange.Begin.Column - 1),
		}
		end := protocol.Position{
			Line:      protocol.UInteger(t.LocRange.End.Line - 1),
			Character: protocol.UInteger(t.LocRange.End.Column - 1),
		}
		variable := protocol.DocumentSymbol{
			Name:  string(t.Id),
			Kind:  protocol.SymbolKindObject,
			Range: protocol.Range{
				Start: start,
				End:   end,
			},
		}
		symbols = append(symbols, variable)
	case *ast.LiteralNull:
	case *ast.SuperIndex:
	}

	return symbols
}
