package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// StructInfo contains information about a struct with encx tags
type StructInfo struct {
	PackageName       string
	StructName        string
	SourceFile        string
	Fields            []FieldInfo
	HasEncxTags       bool
	GenerationOptions map[string]string // From //encx:options comments
}

// FieldInfo contains information about a field with encx tags
type FieldInfo struct {
	Name             string
	Type             string
	EncxTags         []string
	CompanionFields  map[string]CompanionField
	IsValid          bool
	ValidationErrors []string
}


// DiscoveryConfig holds configuration for struct discovery
type DiscoveryConfig struct {
	SkipPackages []string
}

// DiscoverStructs discovers structs with encx tags in the given package path
func DiscoverStructs(packagePath string, config *DiscoveryConfig) ([]StructInfo, error) {
	var structs []StructInfo

	// Parse all .go files in the package
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, packagePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for pkgName, pkg := range pkgs {
		// Skip test packages
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}

		for fileName, file := range pkg.Files {
			structs = append(structs, discoverStructsInFile(fset, fileName, file, pkgName)...)
		}
	}

	return structs, nil
}

// discoverStructsInFile discovers structs in a single file
func discoverStructsInFile(fset *token.FileSet, fileName string, file *ast.File, pkgName string) []StructInfo {
	var structs []StructInfo

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.TypeSpec:
			if structType, ok := node.Type.(*ast.StructType); ok {
				structInfo := analyzeStruct(fset, fileName, pkgName, node.Name.Name, structType)
				if structInfo.HasEncxTags {
					structs = append(structs, structInfo)
				}
			}
		}
		return true
	})

	return structs
}

// analyzeStruct analyzes a struct type for encx tags
func analyzeStruct(fset *token.FileSet, fileName, pkgName, structName string, structType *ast.StructType) StructInfo {
	structInfo := StructInfo{
		PackageName:       pkgName,
		StructName:        structName,
		SourceFile:        filepath.Base(fileName),
		Fields:            []FieldInfo{},
		HasEncxTags:       false,
		GenerationOptions: make(map[string]string),
	}

	// Analyze each field
	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			fieldInfo := analyzeField(name.Name, field)
			if len(fieldInfo.EncxTags) > 0 {
				structInfo.HasEncxTags = true
			}
			structInfo.Fields = append(structInfo.Fields, fieldInfo)
		}
	}

	// Note: Companion field validation removed - code generation creates separate structs

	return structInfo
}

// analyzeField analyzes a single field for encx tags
func analyzeField(fieldName string, field *ast.Field) FieldInfo {
	fieldInfo := FieldInfo{
		Name:             fieldName,
		Type:             getTypeString(field.Type),
		EncxTags:         []string{},
		CompanionFields:  make(map[string]CompanionField),
		IsValid:          true,
		ValidationErrors: []string{},
	}

	// Extract encx tags from struct tags
	if field.Tag != nil {
		tagValue := strings.Trim(field.Tag.Value, "`")
		fieldInfo.EncxTags = extractEncxTags(tagValue)
	}

	// Validate tags if any exist
	if len(fieldInfo.EncxTags) > 0 {
		validator := NewTagValidator()
		errors := validator.ValidateFieldTags(fieldName, fieldInfo.EncxTags)
		if len(errors) > 0 {
			fieldInfo.IsValid = false
			fieldInfo.ValidationErrors = errors
		}
	}

	return fieldInfo
}

// getTypeString converts an ast.Expr to its string representation
func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		return "[]" + getTypeString(t.Elt)
	case *ast.StarExpr:
		return "*" + getTypeString(t.X)
	case *ast.SelectorExpr:
		return getTypeString(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

// extractEncxTags extracts encx tags from a struct tag string
func extractEncxTags(tagString string) []string {
	// Simple implementation - in real code, use reflect.StructTag
	parts := strings.Split(tagString, " ")
	for _, part := range parts {
		if strings.HasPrefix(part, "encx:") {
			// Extract value between quotes
			value := strings.Trim(strings.TrimPrefix(part, "encx:"), "\"")
			return strings.Split(value, ",")
		}
	}
	return []string{}
}