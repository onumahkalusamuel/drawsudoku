package main

import (
	"errors"
	"flag"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type difficultyValue struct {
	Difficulty *string
}

func (d difficultyValue) String() string {
	if d.Difficulty != nil {
		return *d.Difficulty
	}
	return "any"
}

func (d difficultyValue) Set(s string) error {
	difficulty := strings.ToLower(s)
	switch difficulty {
	case "intermediate", "simple", "easy", "expert", "any":
		*d.Difficulty = difficulty
		return nil
	}
	return errors.New("invalid difficulty value")
}

func main() {
	nxPtr := flag.Int("nx", 2, "number of sudokus put horizontally")
	nyPtr := flag.Int("ny", 1, "number of sudokus put vertically")
	nPages := flag.Int("np", 1, "number of sudoku pages to generate")

	difficulty := "any"
	flag.Var(&difficultyValue{&difficulty}, "difficulty", "one of simple, easy, intermediate, expert, any")

	flag.Parse()

	nx := *nxPtr
	ny := *nyPtr
	np := *nPages
	n := nx * ny * np

	fmt.Printf("Generating %d %s Sudokus in a %d x %d grid\n", n, difficulty, nx, ny)

	sudokus := generateSudokus(n, difficulty)

	timestamp := time.Now().Format("20060102-150405")

	filename := fmt.Sprintf("sudokus/sudokus-%v-%dx%d-%s.pdf", timestamp, nx, ny, difficulty)
	createPDF(sudokus, timestamp, nx, ny, filename, np)
}

func generateSudokus(amount int, difficulty string) []string {
	out, err := exec.Command("sh", "-c", fmt.Sprintf("qqwing --generate %d --one-line --difficulty %s", amount, difficulty)).Output()
	if err != nil {
		fmt.Printf("Error: %v", err)
		panic(err)
		// return nil
	}
	fmt.Println(string(out))
	return strings.Split(strings.Trim(string(out), "\n"), "\n")
}

func smaller(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func createPDF(sudokus []string, timestamp string, nx, ny int, filename string, np int) {

	sudokuIndex := 0
	backgroundImage := "4.jpg"
	title := "Killer Sudoku - Volume #5 - Easy"

	pdf := gofpdf.New("L", "mm", "A5", "")
	//  pages for prelim
	pdf.AddPage()
	width, height := pdf.GetPageSize()
	margin := 5. //5 mm

	drawingWidth := width - 5*margin   // 2 for the heading + 1 left + 1 right
	drawingHeight := height - 4*margin // 3 top, 1 bottom

	offsetY := (height - drawingHeight) / 1.2

	L := smaller(drawingWidth/float64(nx), drawingHeight/float64(ny)) * 0.85 //small sudoku length
	fieldL := L / 9

	thinLineWidth := L / 300
	thickLineWidth := L / 120

	// create a few pages with background image
	for v := 0; v < 2; v++ {
		pdf.AddPage()
		pdf.Image(backgroundImage, 0, 0, width, height, false, "", 0, "")
	}

	for W := 0; W < np; W++ {

		pdf.AddPage()
		pdf.Image(backgroundImage, 0, 0, width, height, false, "", 0, "")
		pdf.SetMargins(0, 0, 0)
		pdf.SetAutoPageBreak(false, 0)
		pdf.SetDrawColor(0, 0, 0)

		pdf.SetFont("Helvetica", "", 12)

		//draw title
		pdf.MoveTo(0, height+3*margin)
		pdf.TransformBegin()
		pdf.TransformRotate(90, 0, height)
		pdf.SetFont("Helvetica", "I", 10)
		pdf.CellFormat(height, 2*margin, title, "", 1, "MC", false, 0, "")
		pdf.TransformEnd()

		for X := 0; X < nx; X++ {
			for Y := 0; Y < ny; Y++ {

				x0 := 4*margin + float64(X)/float64(nx)*drawingWidth + (drawingWidth/float64(nx)-L)/2
				y0 := offsetY + float64(Y)/float64(ny)*drawingHeight + (drawingHeight/float64(ny)-L)/2

				// write game number on top
				pdf.SetFont("Helvetica", "IB", fieldL*0.8*2.83)
				pdf.MoveTo(x0, y0-3*margin)
				pdf.CellFormat(L, 3*margin, fmt.Sprintf(" #%d ", sudokuIndex+1), "T", 0, "MC", false, 0, "")

				// set font for sudoku
				pdf.SetFont("Helvetica", "", fieldL*0.8*2.83) //2.83 points is a mm

				// draw horizontal lines
				for ly := 0; ly < 10; ly++ {
					var w float64
					if ly%3 == 0 {
						w = thickLineWidth
					} else {
						w = thinLineWidth
					}
					pdf.SetLineWidth(w)
					pdf.Line(x0-w/2, y0+fieldL*float64(ly), x0+w/2+L, y0+fieldL*float64(ly))
				}
				// draw vertical lines
				for lx := 0; lx < 10; lx++ {
					var w float64
					if lx%3 == 0 {
						w = thickLineWidth
					} else {
						w = thinLineWidth
					}
					pdf.SetLineWidth(w)
					pdf.Line(x0+fieldL*float64(lx), y0-w/2, x0+fieldL*float64(lx), y0+w/2+L)
				}
				// draw numbers
				for i := 0; i < 9; i++ {
					for j := 0; j < 9; j++ {
						n := sudokus[sudokuIndex][i*9+j]
						if string(n) != "." {
							dy := fieldL / 20
							pdf.MoveTo(x0+fieldL*float64(i), y0+fieldL*float64(j)+dy)

							//parameters for drawing the number: cell w, h, number, no borders,
							//don't move, center verically & horizontally, no fill, no link x2
							pdf.CellFormat(fieldL, fieldL, string(n), "", 0, "CM", false, 0, "")
						}
					}
				}
				sudokuIndex++
			}
			// Page number
			pdf.MoveTo(0, height-4*margin)
			pdf.SetFont("Helvetica", "", fieldL*0.8*2.83/1.5)
			pdf.CellFormat(0, 2*margin, fmt.Sprintf("P%d", pdf.PageNo()), "0", 0, "MC", false, 0, "")

		}

	}

	err := pdf.OutputFileAndClose(filename)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Wrote sudokus to file %s\n", filename)
	}
}
