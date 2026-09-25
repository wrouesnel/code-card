package codecard

import (
	"io"

	"codeberg.org/go-pdf/fpdf"
)

const (
	mm = 72.0 / 25.4

	fontFamily = "Courier"
	fontSize   = 12.0
	// Courier ascent and descent from the font's AFM metrics, per 1000 units.
	fontAscent  = 629.0
	fontDescent = -157.0

	// columns is the number of table columns on the card.
	columns = 2
	// strokeAllowance accounts for stroke border widths when centering text in a row.
	strokeAllowance = 2.0
)

// point is a coordinate in PDF points, with the origin at the bottom left of the page.
type point struct{ X, Y float64 }

func (p point) Add(o point) point       { return point{p.X + o.X, p.Y + o.Y} }
func (p point) Sub(o point) point       { return point{p.X - o.X, p.Y - o.Y} }
func (p point) Mul(scale float64) point { return point{p.X * scale, p.Y * scale} }

//nolint:gochecknoglobals
var (
	pageSize     = point{56.0 * mm, 85.6 * mm}
	borderMargin = point{1 * mm, 1 * mm}
	// Padding is intentionally lop-sided since sleeves cut the bottom off by more.
	paddingLB = point{5 * mm, 13 * mm}
	paddingRT = point{5 * mm, 1 * mm}
)

// RenderOptions control card rendering.
type RenderOptions struct {
	// Title is drawn at the top of the card if not empty.
	Title string
	// DebugGrid draws the usable area and table cell boundaries.
	DebugGrid bool
}

// canvas wraps fpdf with a bottom-left origin coordinate system.
type canvas struct {
	*fpdf.Fpdf
}

func (c canvas) rect(origin, size point) {
	c.Rect(origin.X, pageSize.Y-origin.Y-size.Y, size.X, size.Y, "D")
}

func (c canvas) line(from, to point) {
	c.Line(from.X, pageSize.Y-from.Y, to.X, pageSize.Y-to.Y)
}

// drawString draws text with its baseline starting at pos.
func (c canvas) drawString(pos point, text string) {
	c.Text(pos.X, pageSize.Y-pos.Y, text)
}

// Render writes table as a single-page credit-card sized PDF to w.
func Render(w io.Writer, table *Table, opts RenderOptions) error {
	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "pt",
		Size:           fpdf.SizeType{Wd: pageSize.X, Ht: pageSize.Y},
	})
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetMargins(0, 0, 0)
	pdf.AddPage()
	pdf.SetFont(fontFamily, "", fontSize)
	pdf.SetLineWidth(1)
	c := canvas{pdf}

	cutBorder := [2]point{borderMargin, pageSize.Sub(borderMargin.Mul(2))}
	usableRect := [2]point{
		borderMargin.Add(paddingLB),
		pageSize.Sub(paddingLB).Sub(borderMargin.Mul(2).Add(paddingRT)),
	}
	usableTop := usableRect[0].Y + usableRect[1].Y

	// Draw the cut border
	c.rect(cutBorder[0], cutBorder[1])

	if opts.DebugGrid {
		c.rect(usableRect[0], usableRect[1])
	}

	rows := table.Rows()
	dataRows := (len(rows) + columns - 1) / columns
	totalRows := dataRows + 1 // Add one more row for the title

	columnWidth := usableRect[1].X / columns
	rowHeight := usableRect[1].Y / float64(totalRows)

	fontHeight := (fontAscent - fontDescent) / 1000 * fontSize
	rowCenterOffset := rowHeight - (rowHeight - fontHeight - strokeAllowance)

	if opts.Title != "" {
		titleWidth := pdf.GetStringWidth(opts.Title)
		c.drawString(point{
			usableRect[0].X + usableRect[1].X/2 - titleWidth/2,
			usableTop - rowCenterOffset,
		}, opts.Title)
	}

	c.line(point{usableRect[0].X, usableTop - rowHeight}, point{usableRect[0].X + usableRect[1].X, usableTop - rowHeight})

	for idx, text := range rows {
		rowIdx := float64(idx/columns + 1)
		colIdx := float64(idx % columns)

		if opts.DebugGrid {
			c.rect(point{
				usableRect[0].X + columnWidth*colIdx,
				usableTop - rowHeight - rowHeight*rowIdx,
			}, point{columnWidth, rowHeight})
		}

		textWidth := pdf.GetStringWidth(text)
		c.drawString(point{
			usableRect[0].X + usableRect[1].X/(2*columns) + columnWidth*colIdx - textWidth/2,
			usableTop - rowIdx*rowHeight - rowCenterOffset,
		}, text)
	}

	return pdf.Output(w)
}
