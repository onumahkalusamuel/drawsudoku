package internal

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os/exec"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type difficultyValue struct {
	Difficulty *string
}

func (d difficultyValue) String() string {
	if d.Difficulty != nil {
		return *d.Difficulty
	}
	return "simple"
}

func (d difficultyValue) Set(s string) error {
	difficulty := strings.ToLower(s)
	switch difficulty {
	case "intermediate", "simple", "easy", "expert":
		*d.Difficulty = difficulty
		return nil
	}
	return errors.New("invalid difficulty value")
}

func main() {

	nums := flag.Int("nums", 100, "number of sudokus to generate at a time")

	difficulty := "any"
	flag.Var(&difficultyValue{&difficulty}, "difficulty", "one of simple, easy, intermediate, expert, any")

	flag.Parse()

	n := *nums

	fmt.Printf("Generating %d %s Sudokus\n", n, difficulty)

	out, err := exec.Command("sh", "-c", fmt.Sprintf("qqwing --generate %d --one-line --difficulty %s", n, difficulty)).Output()
	if err != nil {
		fmt.Printf("Error: %v", err)
		panic(err)
		// return nil
	}
	fmt.Println(string(out))

	holder := strings.Trim(string(out), "\n")

	games := strings.Split(holder, "\n")

	// generate the results
	res, err := exec.Command("sh", "-c", "echo \""+holder+"\" | qqwing --solve --one-line").Output()

	if err != nil {
		fmt.Printf("Error: %v", err)
		panic(err)
		// return nil
	}

	results := strings.Split(strings.Trim(string(res), "\n"), "\n")

	// run the db stuffs
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/sudoku")

	// if there is an error opening the connection, handle it
	if err != nil {
		panic(err.Error())
	}

	// defer the close till after the main function has finished
	// executing
	defer db.Close()

	// store results
	for Y := 0; Y < n; Y++ {

		// check if value already exists
		read, err := db.Query("SELECT id FROM sudoku_"+difficulty+" WHERE game=?", games[Y])
		if err != nil {
			panic(err.Error())
		}

		// means there's no previous record
		if read.Next() == false {
			// // perform a db.Query insert
			insert, err := db.Query("INSERT INTO sudoku_"+difficulty+"(game, solution) VALUE(?, ?);", games[Y], results[Y])

			// // if there is an error inserting, handle it
			if err != nil {
				panic(err.Error())
			}
			// be careful deferring Queries if you are using transactions
			insert.Close()
		}
	}
}
