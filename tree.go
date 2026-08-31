package mf4

import (
	"fmt"
	"os"
	"sync"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/datasection"
)

func osOpen(path string) (*os.File, error) { return os.Open(path) }

// buildTree decodes the DG → CG → CN metadata tree.
func (f *File) buildTree() error {
	dgIndex := 0
	for dga := f.hd.DGFirst; dga != 0; dgIndex++ {
		dg, err := blocks.DecodeDG(f.src, dga)
		if err != nil {
			return err
		}
		dgState := &dataGroup{file: f, block: dg, index: dgIndex}
		for cga := dg.CGFirst; cga != 0; {
			cg, err := blocks.DecodeCG(f.src, cga)
			if err != nil {
				return err
			}
			if cg.IsVLSD() {
				// VLSD service group: a container for variable-length
				// data, addressed by record ID from a channel's cn_data
				// link. Not a user-facing group.
				dgState.vlsdGroups = append(dgState.vlsdGroups, cg)
			} else {
				g, err := f.buildGroup(dgState, cg)
				if err != nil {
					return err
				}
				f.groups = append(f.groups, g)
				f.channels = append(f.channels, g.channels...)
			}
			dgState.groups = append(dgState.groups, cg)
			cga = cg.CGNext
		}
		dga = dg.DGNext
	}
	return nil
}

// dataGroup is the internal per-DG state shared by the channel groups it
// contains (record layout, de-interleave cache for unsorted files).
type dataGroup struct {
	file  *File
	block *blocks.DG
	index int

	groups     []*blocks.CG // all CGs in this DG, VLSD included
	vlsdGroups []*blocks.CG

	sectionOnce sync.Once
	section     *datasection.Reader
	sectionErr  error

	// derivedIDs maps the record IDs of CG-template array elements —
	// which have no CGBLOCK of their own — to their record size.
	derivedIDs map[uint64]int

	// De-interleave state for unsorted files: one record stream per
	// record ID, built lazily on first read.
	dinOnce sync.Once
	dinBufs map[uint64]*datasection.Reader
	dinErr  error
}

func (f *File) buildGroup(dg *dataGroup, cg *blocks.CG) (*ChannelGroup, error) {
	g := &ChannelGroup{
		RecordCount: cg.CycleCount,
		file:        f,
		dg:          dg,
		cg:          cg,
	}
	var err error
	if g.Name, err = blocks.DecodeText(f.src, cg.TXAcqName); err != nil {
		return nil, err
	}
	if g.Comment, err = blocks.CommentText(f.src, cg.MDComment); err != nil {
		return nil, err
	}
	if g.Source, err = f.sourceInfo(cg.SIAcqSource); err != nil {
		return nil, err
	}
	if err := f.buildChannels(g, cg.CNFirst, nil); err != nil {
		return nil, err
	}
	return g, nil
}

// buildChannels walks a CN chain (and recursively any structure
// composition chains) appending to g.channels. parent is non-nil for
// component channels of a structure.
func (f *File) buildChannels(g *ChannelGroup, first int64, parent *Channel) error {
	for cna := first; cna != 0; {
		cn, err := blocks.DecodeCN(f.src, cna)
		if err != nil {
			return err
		}
		c := &Channel{
			Type:     ChannelType(cn.Type),
			cnAddr:   cna,
			DataType: DataType(cn.DataType),
			BitCount: cn.BitCount,
			group:    g,
			cn:       cn,
			parent:   parent,
		}
		if c.Name, err = blocks.DecodeText(f.src, cn.TXName); err != nil {
			return err
		}
		if c.Unit, err = blocks.CommentText(f.src, cn.MDUnit); err != nil {
			return err
		}
		if c.Comment, err = blocks.CommentText(f.src, cn.MDComment); err != nil {
			return err
		}
		if c.Source, err = f.sourceInfo(cn.SISource); err != nil {
			return err
		}
		if c.cc, err = blocks.DecodeCC(f.src, cn.CCConversion); err != nil {
			return err
		}
		if c.Unit == "" && c.cc != nil {
			if c.Unit, err = blocks.CommentText(f.src, c.cc.MDUnit); err != nil {
				return err
			}
		}
		c.Conversion, err = f.conversionInfo(c.cc)
		if err != nil {
			return err
		}
		if cn.Type == blocks.CNMaster || cn.Type == blocks.CNVirtualMaster {
			g.master = c
		}
		g.channels = append(g.channels, c)

		// Composition: either a nested CN chain (structure) whose
		// components are ordinary channels, or a CA block (array).
		if cn.Composition != 0 {
			id, err := blocks.PeekID(f.src, cn.Composition)
			if err != nil {
				return err
			}
			switch id {
			case blocks.IDCN:
				if err := f.buildChannels(g, cn.Composition, c); err != nil {
					return err
				}
			case blocks.IDCA:
				c.IsArray = true
				if err := f.expandArray(g, c, cn.Composition); err != nil {
					return err
				}
			}
		}
		cna = cn.CNNext
	}
	return nil
}

func (f *File) sourceInfo(addr int64) (SourceInfo, error) {
	si, err := blocks.DecodeSI(f.src, addr)
	if err != nil || si == nil {
		return SourceInfo{}, err
	}
	s := SourceInfo{Type: SourceType(si.Type), Bus: BusType(si.BusType)}
	if s.Name, err = blocks.DecodeText(f.src, si.TXName); err != nil {
		return s, err
	}
	if s.Path, err = blocks.DecodeText(f.src, si.TXPath); err != nil {
		return s, err
	}
	if s.Comment, err = blocks.CommentText(f.src, si.MDComment); err != nil {
		return s, err
	}
	return s, nil
}

// expandArray adds one scalar element channel per array element for a
// channel whose composition is a CA block (or a chain of nested CA
// blocks) with CN-template storage — the layout used by measurement
// arrays, maps and classification results. Element channels are named
// name[i]...[j] (row-major, last dimension fastest), and their byte
// offsets follow the spec formula: base + sum(index_d * stride_d) with
// the stride of a level's last dimension equal to that level's
// ca_byte_offset_base.
func (f *File) expandArray(g *ChannelGroup, parent *Channel, caAddr int64) error {
	type dim struct {
		size      uint64
		stride    int64
		invStride uint32
	}
	var dims []dim
	total := uint64(1)
	for addr := caAddr; addr != 0; {
		ca, err := blocks.DecodeCA(f.src, addr)
		if err != nil {
			return err
		}
		if len(ca.DimSize) == 0 {
			return nil
		}
		if ca.Storage != blocks.CAStorageCNTemplate {
			// "Fragmented" array: each element is recorded in its own
			// records (own record ID or own data section), all sharing
			// the parent's record layout.
			return f.expandFragmentedArray(g, parent, ca)
		}
		// Within one level: row-major, last dimension has stride
		// ca_byte_offset_base, earlier dimensions multiply up.
		stride := int64(ca.ByteOffsetBase)
		inv := ca.InvalBitPosBase
		level := make([]dim, len(ca.DimSize))
		for d := len(ca.DimSize) - 1; d >= 0; d-- {
			level[d] = dim{size: ca.DimSize[d], stride: stride, invStride: inv}
			stride *= int64(ca.DimSize[d])
			inv *= uint32(ca.DimSize[d])
		}
		dims = append(dims, level...)
		for _, dm := range level {
			total *= dm.size
			if total == 0 || total > 1<<20 {
				return nil // degenerate or absurd; keep the raw column only
			}
		}
		if ca.Composition != 0 {
			id, err := blocks.PeekID(f.src, ca.Composition)
			if err != nil || id != blocks.IDCA {
				return err // CN composition below a CA: not expanded
			}
		}
		addr = ca.Composition
	}

	idx := make([]uint64, len(dims))
	for k := uint64(0); k < total; k++ {
		rem := k
		for d := len(dims) - 1; d >= 0; d-- {
			idx[d] = rem % dims[d].size
			rem /= dims[d].size
		}
		name := parent.Name
		byteOff := int64(parent.cn.ByteOffset)
		invOff := uint32(0)
		for d, i := range idx {
			name += fmt.Sprintf("[%d]", i)
			byteOff += int64(i) * dims[d].stride
			invOff += uint32(i) * dims[d].invStride
		}
		cnCopy := *parent.cn
		cnCopy.ByteOffset = uint32(byteOff)
		if cnCopy.Flags&blocks.CNFlagInvalBit != 0 {
			cnCopy.InvalBitPos = parent.cn.InvalBitPos + invOff
		}
		cnCopy.Composition = 0
		elem := &Channel{
			Name:       name,
			Unit:       parent.Unit,
			Comment:    parent.Comment,
			Source:     parent.Source,
			Type:       parent.Type,
			DataType:   parent.DataType,
			BitCount:   parent.BitCount,
			Conversion: parent.Conversion,
			group:      g,
			parent:     parent,
			cn:         &cnCopy,
			cc:         parent.cc,
		}
		g.channels = append(g.channels, elem)
	}
	return nil
}

// fixCycleCounts recomputes each group's record count from the actual
// data (finalization step for unfinalized files whose cycle counters
// were never written).
func (f *File) fixCycleCounts() error {
	for _, g := range f.groups {
		if g.cg.RecordSize() == 0 {
			continue
		}
		layout, err := g.recordLayout()
		if err != nil {
			return err
		}
		g.RecordCount = uint64(layout.data.Size() / int64(layout.recSize))
		g.cg.CycleCount = g.RecordCount
	}
	return nil
}

// expandFragmentedArray adds one element channel per array element for
// CG-template (own record ID in an unsorted group) and DG-template (own
// data section) storage. All elements share the parent's record layout,
// so only the record source and cycle count differ.
func (f *File) expandFragmentedArray(g *ChannelGroup, parent *Channel, ca *blocks.CA) error {
	total := ca.ElementCount()
	if total <= 1 || total > 1<<20 {
		return nil
	}
	if ca.Storage == blocks.CAStorageDGTemplate && uint64(len(ca.DataLinks)) < total {
		return nil
	}
	idx := make([]uint64, len(ca.DimSize))
	for k := uint64(0); k < total; k++ {
		rem := k
		for d := len(ca.DimSize) - 1; d >= 0; d-- {
			idx[d] = rem % ca.DimSize[d]
			rem /= ca.DimSize[d]
		}
		name := parent.Name
		for _, i := range idx {
			name += fmt.Sprintf("[%d]", i)
		}
		elem := &arrayElement{recordCount: g.RecordCount}
		if int(k) < len(ca.CycleCounts) {
			elem.recordCount = ca.CycleCounts[k]
		}
		if ca.Storage == blocks.CAStorageDGTemplate {
			elem.dataAddr = ca.DataLinks[k]
			if elem.dataAddr == 0 {
				continue // element was not recorded at all
			}
		} else {
			elem.recordID = g.cg.RecordID + k
			if g.dg.derivedIDs == nil {
				g.dg.derivedIDs = make(map[uint64]int)
			}
			g.dg.derivedIDs[elem.recordID] = int(g.cg.RecordSize())
		}
		cnCopy := *parent.cn
		cnCopy.Composition = 0
		g.channels = append(g.channels, &Channel{
			Name:       name,
			Unit:       parent.Unit,
			Comment:    parent.Comment,
			Source:     parent.Source,
			Type:       parent.Type,
			DataType:   parent.DataType,
			BitCount:   parent.BitCount,
			Conversion: parent.Conversion,
			group:      g,
			parent:     parent,
			elem:       elem,
			cn:         &cnCopy,
			cc:         parent.cc,
		})
	}
	return nil
}
