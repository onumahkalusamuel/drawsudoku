package internal

import (
	"database/sql"
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

func main() {
	volume := flag.Int("volume", 1, "volume number. 1 for first 100, 2 for 101 to 200")
	flag.Parse()

	nx := 1
	ny := 2
	v := *volume
	levels := [4]string{"simple", "easy", "intermediate", "expert"}

	sudokus := fetchSudokuGames(v, levels)

	timestamp := time.Now().Format("20060102-150405")

	filename := fmt.Sprintf("sudokus/sudokus-%v-%dx%d-%s-vol-%d.pdf", timestamp, nx, ny, "mix", v)
	createPDF(sudokus, nx, ny, v, levels, filename)
}

func fetchSudokuGames(volume int, levels [4]string) [][]Game {

	var results = make([][]Game, 4)
	multipier := 50
	basesize := [4]int{1, 1, 3, 6}
	// run the db stuffs
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/sudoku")

	// if there is an error opening the connection, handle it
	if err != nil {
		panic(err.Error())
	}

	// defer the close till after the main function has finished
	// executing
	defer db.Close()

	for i := 0; i < 4; i++ {
		// lets fetch
		offset := 0
		limit := basesize[i] * multipier
		difficulty := levels[i]
		results[i] = make([]Game, (limit - offset))

		if volume > 1 {
			offset = ((volume - 1) * basesize[i] * multipier) + 1
		}

		read, err := db.Query("SELECT game, solution FROM sudoku_"+difficulty+" LIMIT ? OFFSET ?", limit, offset)
		if err != nil {
			panic(err.Error())
		}

		pointer := 0

		for read.Next() {
			var game Game

			err = read.Scan(&game.game, &game.solution)
			if err != nil {
				panic(err.Error()) // proper error handling instead of panic in your app
			}

			results[i][pointer] = game

			pointer++
		}
	}

	return results
}

func smaller(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func createPDF(sudokus [][]Game, nx, ny int, volume int, levels [4]string, filename string) {

	pdf := gofpdf.New("P", "mm", "Letter", "")
	// prelim pages
	pdf.AddPage()
	pdf.AddPage()
	pdf.AddPage()
	pdf.AddPage()

	// continue
	width, height := pdf.GetPageSize()
	margin := 6. //6 mm
	basenx := nx
	baseny := ny
	sudokuIndex := 0

	// let's get this looping
	for K := 0; K < 4; K++ {

		difficulty := strings.Title(levels[K])
		count := len(sudokus[K])

		nx = basenx
		ny = baseny

		sudokuIndex = 0

		drawingWidth := width - 6*margin
		drawingHeight := height - 6*margin

		offsetY := (height - drawingHeight) / 2.5

		L := smaller(drawingWidth/float64(nx), drawingHeight/float64(ny)) * 0.85 //small sudoku length
		fieldL := L / 9

		thinLineWidth := L / 300
		thickLineWidth := L / 120

		pdf.AddPage()
		pdf.SetMargins(0, 0, 0)
		pdf.SetAutoPageBreak(false, 0)
		pdf.SetDrawColor(0, 0, 0)

		//draw title
		pdf.MoveTo(0, 0)
		pdf.SetFont("Helvetica", "B", 24)
		pdf.CellFormat(width, height, fmt.Sprint(difficulty, " Sudoku - Puzzles"), "", 1, "MC", false, 0, "")
		pdf.CellFormat(width, height, fmt.Sprint("Volume #", volume), "", 1, "MC", false, 0, "")

		np := count / 2

		for W := 0; W < np; W++ {

			pdf.AddPage()
			pdf.SetMargins(0, 0, 0)
			pdf.SetAutoPageBreak(false, 0)
			pdf.SetDrawColor(0, 0, 0)

			for X := 0; X < nx; X++ {
				for Y := 0; Y < ny; Y++ {

					x0 := 3*margin + float64(X)/float64(nx)*drawingWidth + (drawingWidth/float64(nx)-L)/2
					y0 := offsetY + float64(Y)/float64(ny)*drawingHeight + (drawingHeight/float64(ny)-L)/2

					// write game number on top
					pdf.SetFont("Helvetica", "B", fieldL*0.7*2.83)
					pdf.MoveTo(x0, y0-2*margin)
					pdf.CellFormat(L, 2*margin, fmt.Sprintf("%s Sudoku - #%d ", difficulty, sudokuIndex+1), "", 0, "MC", false, 0, "")

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
							n := sudokus[K][sudokuIndex].game[i*9+j]
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
				pdf.MoveTo(0, height-(3.5*margin))
				pdf.SetFont("Helvetica", "", 14)
				pdf.CellFormat(0, 2*margin, fmt.Sprintf("%d", pdf.PageNo()), "0", 0, "MC", false, 0, "")
			}

		}

		// populate answer pages
		np = count / 6
		if (count % 6) != 0 {
			np += 1
		}

		nx = 2
		ny = 3

		sudokuIndex = 0

		drawingWidth = width - 6*margin
		drawingHeight = height - 6*margin

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
		pdf.CellFormat(width, height, fmt.Sprint(difficulty, " Sudoku - Solutions"), "", 1, "MC", false, 0, "")

		// main solutions
		for W := 0; W < np; W++ {

			pdf.AddPage()
			pdf.SetMargins(0, 0, 0)
			pdf.SetAutoPageBreak(false, 0)
			pdf.SetDrawColor(0, 0, 0)

			for X := 0; X < nx; X++ {
				for Y := 0; Y < ny; Y++ {

					if sudokuIndex >= count {
						break
					}
					x0 := 3*margin + float64(X)/float64(nx)*drawingWidth + (drawingWidth/float64(nx)-L)/2
					y0 := offsetY + float64(Y)/float64(ny)*drawingHeight + (drawingHeight/float64(ny)-L)/2

					// write game number on top
					pdf.SetFont("Helvetica", "B", fieldL*0.7*2.83)
					pdf.MoveTo(x0, y0-1*margin)
					pdf.CellFormat(L, 1*margin, fmt.Sprintf("%s Sudoku - #%d", strings.Title(difficulty), sudokuIndex+1), "", 0, "MC", false, 0, "")

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
							n := sudokus[K][sudokuIndex].solution[i*9+j]
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
				pdf.MoveTo(0, height-(3.5*margin))
				pdf.SetFont("Helvetica", "", 14)
				pdf.CellFormat(0, 2*margin, fmt.Sprintf("%d", pdf.PageNo()), "0", 0, "MC", false, 0, "")
			}

		}
	}

	err := pdf.OutputFileAndClose(filename)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Wrote sudokus to file %s\n", filename)
	}
}
