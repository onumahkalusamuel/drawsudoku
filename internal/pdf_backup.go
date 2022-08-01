package internal

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jung-kurt/gofpdf"
)

type Game struct {
	game     string `json:"game"`
	solution string `json:"solution"`
}

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

type paperSizeValue struct {
	PaperSize *string
}

func (d paperSizeValue) String() string {
	if d.PaperSize != nil {
		return *d.PaperSize
	}
	return "Letter"
}

func (d paperSizeValue) Set(s string) error {
	paperSize := strings.Title(s)
	switch paperSize {
	case "A4", "A5", "Letter":
		*d.PaperSize = paperSize
		return nil
	}
	return errors.New("invalid paper size")
}

type orientationValue struct {
	Orientation *string
}

func (d orientationValue) String() string {
	if d.Orientation != nil {
		return *d.Orientation
	}
	return "P"
}

func (d orientationValue) Set(s string) error {
	orientation := strings.ToUpper(s)
	switch orientation {
	case "L", "P":
		*d.Orientation = orientation
		return nil
	}
	return errors.New("invalid orientation value")
}

func main() {
	volume := flag.Int("volume", 1, "volume number. 1 for first 100, 2 for 101 to 200")
	count := flag.Int("count", 100, "number of sudoku games to fetch")

	paperSize := "Letter"
	flag.Var(&paperSizeValue{&paperSize}, "papersize", "one of A4, A5, Letter")

	orientation := "P"
	flag.Var(&orientationValue{&orientation}, "orientation", "one of L (for landscape), P (for portrait)")

	difficulty := "any"
	flag.Var(&difficultyValue{&difficulty}, "difficulty", "one of simple, easy, intermediate, expert, any")

	flag.Parse()

	nx := 1
	ny := 1
	n := *count
	v := *volume

	if orientation == "L" {
		nx = 2
	}

	if orientation == "P" {
		ny = 2
	}

	fmt.Printf("Creating %d %s Sudokus in a %d x %d grid\n", n, difficulty, nx, ny)

	sudokus := fetchSudokuGames(n, difficulty, v)

	timestamp := time.Now().Format("20060102-150405")

	filename := fmt.Sprintf("sudokus/sudokus-%v-%dx%d-%s.pdf", timestamp, nx, ny, difficulty)
	createPDF(sudokus, nx, ny, n, v, difficulty, orientation, filename, paperSize)
}

func fetchSudokuGames(amount int, difficulty string, volume int) []Game {

	var results = make([]Game, amount)
	pointer := 0
	// run the db stuffs
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/sudoku")

	// if there is an error opening the connection, handle it
	if err != nil {
		panic(err.Error())
	}

	// defer the close till after the main function has finished
	// executing
	defer db.Close()

	// lets fetch
	offset := 0
	limit := amount

	if volume > 1 {
		offset = (volume * 100) + 1
	}

	read, err := db.Query("SELECT game, solution FROM sudoku_"+difficulty+" LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		panic(err.Error())
	}

	for read.Next() {
		var game Game

		err = read.Scan(&game.game, &game.solution)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		results[pointer] = game
		pointer++
	}

	return results
}

func smaller(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func createPDF(sudokus []Game, nx, ny int, count int, volume int, difficulty string, orientation string, filename string, paperSize string) {

	sudokuIndex := 0

	title := strings.Title(fmt.Sprintf(difficulty+" Sudoku - Volume #%d - ZebiGames", volume))

	pdf := gofpdf.New(orientation, "mm", paperSize, "")
	//  pages for prelim
	// pdf.AddPage()
	width, height := pdf.GetPageSize()
	margin := 6. //6 mm

	drawingWidth := width - 5*margin
	drawingHeight := height - 6*margin

	offsetY := (height - drawingHeight) / 2

	L := smaller(drawingWidth/float64(nx), drawingHeight/float64(ny)) * 0.85 //small sudoku length
	fieldL := L / 9

	thinLineWidth := L / 300
	thickLineWidth := L / 120

	//draw title
	pdf.MoveTo(0, 0)
	pdf.SetFont("Helvetica", "B", 24)
	pdf.CellFormat(width, height, "Puzzles", "", 1, "MC", false, 0, "")

	np := count / 2

	for W := 0; W < np; W++ {

		pdf.AddPage()
		pdf.SetMargins(0, 0, 0)
		pdf.SetAutoPageBreak(false, 0)
		pdf.SetDrawColor(0, 0, 0)

		pdf.SetFont("Helvetica", "", 12)

		//draw title
		pdf.MoveTo(0, height+2.5*margin)
		pdf.TransformBegin()
		pdf.TransformRotate(90, 0, height)
		pdf.SetFont("Helvetica", "I", 10)
		pdf.CellFormat(height, 2*margin, title, "", 1, "MC", false, 0, "")
		pdf.TransformEnd()

		for X := 0; X < nx; X++ {
			for Y := 0; Y < ny; Y++ {

				x0 := 3*margin + float64(X)/float64(nx)*drawingWidth + (drawingWidth/float64(nx)-L)/2
				y0 := offsetY + float64(Y)/float64(ny)*drawingHeight + (drawingHeight/float64(ny)-L)/2

				// write game number on top
				pdf.SetFont("Helvetica", "B", fieldL*0.8*2.83)
				pdf.MoveTo(x0, y0-2*margin)
				pdf.CellFormat(L, 2*margin, fmt.Sprintf("Sudoku - %s #%d ", strings.Title(difficulty), sudokuIndex+1), "", 0, "MC", false, 0, "")

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
						n := sudokus[sudokuIndex].game[i*9+j]
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
			pdf.MoveTo(0, height-3*margin)
			pdf.SetFont("Helvetica", "", fieldL*0.8*2.83/1.5)
			pdf.CellFormat(0, 2*margin, fmt.Sprintf("P%d", pdf.PageNo()), "0", 0, "MC", false, 0, "")

		}

	}

	// populate answer pages
	np = count / 6
	if (count % 6) != 0 {
		np += 1
	}

	nx = 2
	ny = 2

	if orientation == "L" {
		nx = 3
	}
	if orientation == "P" {
		ny = 3
	}

	sudokuIndex = 0

	drawingWidth = width - 6*margin   // 2 for the heading + 1 left + 1 right
	drawingHeight = height - 5*margin // 3 top, 1 bottom

	offsetY = (height - drawingHeight) / 2.5

	L = smaller(drawingWidth/float64(nx), drawingHeight/float64(ny)) * 0.85 //small sudoku length
	fieldL = L / 9

	thinLineWidth = L / 300
	thickLineWidth = L / 120

	// solutions title
	pdf.AddPage()
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetDrawColor(0, 0, 0)

	//draw title
	pdf.MoveTo(0, 0)
	pdf.SetFont("Helvetica", "B", 24)
	pdf.CellFormat(width, height, "Solutions", "", 1, "MC", false, 0, "")

	// main solutions
	for W := 0; W < np; W++ {

		pdf.AddPage()
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

				if sudokuIndex >= count {
					break
				}
				x0 := 4*margin + float64(X)/float64(nx)*drawingWidth + (drawingWidth/float64(nx)-L)/2
				y0 := offsetY + float64(Y)/float64(ny)*drawingHeight + (drawingHeight/float64(ny)-L)/2

				// write game number on top
				pdf.SetFont("Helvetica", "IB", fieldL*0.8*2.83)
				pdf.MoveTo(x0, y0-1*margin)
				pdf.CellFormat(L, 1*margin, fmt.Sprintf(" #%d ", sudokuIndex+1), "", 0, "MC", false, 0, "")

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
					// fmt.Println(sudokuIndex)
					for j := 0; j < 9; j++ {
						n := sudokus[sudokuIndex].solution[i*9+j]
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
			pdf.MoveTo(0, height-3*margin)
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
