package mf4

import (
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
