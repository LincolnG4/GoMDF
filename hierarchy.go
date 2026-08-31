package mf4

import (
	"github.com/LincolnG4/GoMDF/internal/blocks"
)

// HierarchyType classifies a channel hierarchy node (ch_type).
type HierarchyType uint8

const (
	HierarchyGroup      HierarchyType = blocks.CHGroup
	HierarchyFunction   HierarchyType = blocks.CHFunction
	HierarchyStructure  HierarchyType = blocks.CHStructure
	HierarchyMapList    HierarchyType = blocks.CHMapList
	HierarchyInputVars  HierarchyType = blocks.CHInputVars
	HierarchyOutputVars HierarchyType = blocks.CHOutputVars
	HierarchyLocalVars  HierarchyType = blocks.CHLocalVars
	HierarchyCalDefVars HierarchyType = blocks.CHCalDefVars
	HierarchyCalRefVars HierarchyType = blocks.CHCalRefVars
)

// HierarchyNode is one node of the file's logical channel tree (CH
// blocks): a named grouping of channels and child nodes, as defined by
// e.g. MCD-2 MC GROUP/FUNCTION structures.
type HierarchyNode struct {
	Name     string
	Comment  string
	Type     HierarchyType
	Channels []*Channel
	Children []*HierarchyNode
}

// Hierarchy returns the roots of the logical channel hierarchy, or nil
// when the file defines none. Channel references that cannot be resolved
// to decoded channels are skipped.
func (f *File) Hierarchy() ([]*HierarchyNode, error) {
	// Index decoded channels by CN block address for reference lookup.
	byAddr := make(map[int64]*Channel, len(f.channels))
	for _, c := range f.channels {
		byAddr[c.cnAddr] = c
	}
	return f.hierarchyChain(f.hd.CHFirst, byAddr, 0)
}

const maxHierarchyDepth = 64

func (f *File) hierarchyChain(addr int64, byAddr map[int64]*Channel, depth int) ([]*HierarchyNode, error) {
	if depth > maxHierarchyDepth {
		return nil, nil
	}
	var out []*HierarchyNode
	for addr != 0 {
		ch, err := blocks.DecodeCH(f.src, addr)
		if err != nil {
			return nil, err
		}
		node := &HierarchyNode{Type: HierarchyType(ch.Type)}
		if node.Name, err = blocks.DecodeText(f.src, ch.TXName); err != nil {
			return nil, err
		}
		if node.Comment, err = blocks.CommentText(f.src, ch.MDComment); err != nil {
			return nil, err
		}
		for _, el := range ch.Elements {
			if c, ok := byAddr[el.CN]; ok {
				node.Channels = append(node.Channels, c)
			}
		}
		if node.Children, err = f.hierarchyChain(ch.CHFirst, byAddr, depth+1); err != nil {
			return nil, err
		}
		out = append(out, node)
		addr = ch.CHNext
	}
	return out, nil
}
