package indicator

import (
	"errors"
	"fmt"
	"os"

	"github.com/pplcc/plotext"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

type DebugPlot struct {
	plots   []*plot.Plot
	heights []float64
	w       int
	h       int
}

func NewDebugPlot(w, h int) *DebugPlot {
	return &DebugPlot{w: w, h: h}
}

func (d *DebugPlot) Add(p *plot.Plot, height float64) {
	d.plots = append(d.plots, p)
	d.heights = append(d.heights, height)
}

func (d *DebugPlot) Save(path string) (err error) {
	var axis []*plot.Axis
	for _, p := range d.plots {
		axis = append(axis, &p.X)
	}
	plotext.UniteAxisRanges(axis)

	tbl := plotext.Table{
		RowHeights: d.heights,
		ColWidths:  []float64{1},
	}

	var plots2d [][]*plot.Plot
	for _, p := range d.plots {
		plots2d = append(plots2d, []*plot.Plot{p})
	}

	h := 0.0
	for _, v := range d.heights {
		h += v * float64(d.h)
	}

	img := vgimg.New(vg.Points(float64(d.w)), vg.Points(float64(h)))
	dc := draw.New(img)

	canvases := tbl.Align(plots2d, dc)
	for i, p := range d.plots {
		p.Draw(canvases[i][0])
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create plot file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close plot file: %w", err))
		}
	}()

	png := vgimg.PngCanvas{Canvas: img}
	if _, err := png.WriteTo(f); err != nil {
		return fmt.Errorf("failed to write plot to file: %w", err)
	}

	return nil
}
