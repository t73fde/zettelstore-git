//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package zettelmark provides a parser for zettelmarkup.
package zettelmark

import (
	"zettelstore.de/z/ast"
)

// Internal nodes for parsing zettelmark. These will be removed in
// post-processing.

// nullItemNode specifies a removable placeholder for an item block.
type nullItemNode struct {
	ast.ItemNode
}

func (nn *nullItemNode) blockNode() {}
func (nn *nullItemNode) itemNode()  {}

// Accept a visitor and visit the node.
func (nn *nullItemNode) Accept(v ast.Visitor) {}

// nullDescriptionNode specifies a removable placeholder.
type nullDescriptionNode struct {
	ast.DescriptionNode
}

func (nn *nullDescriptionNode) blockNode()       {}
func (nn *nullDescriptionNode) descriptionNode() {}

// Accept a visitor and visit the node.
func (nn *nullDescriptionNode) Accept(v ast.Visitor) {}
