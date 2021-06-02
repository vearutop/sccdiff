package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"code.cloudfoundry.org/bytefmt"
	"github.com/bool64/dev/version"
	"github.com/boyter/scc/v3/processor"
	"github.com/olekukonko/tablewriter"
)

type flags struct {
	baseRef string
	baseDir string
	all     bool
	version bool
}

func main() {
	f := flags{}

	flag.StringVar(&f.baseRef, "baseref", "HEAD", "Base reference.")
	flag.StringVar(&f.baseDir, "basedir", "", "Base directory.")
	flag.BoolVar(&f.all, "all", false, "Include unmodified records in report.")
	flag.BoolVar(&f.version, "version", false, "Show app version and exit.")
	flag.Parse()

	if f.version {
		fmt.Println(version.Info().Version)

		return
	}

	processor.Format = "csv"
	processor.MinifiedGenerated = true
	processor.Minified = true
	processor.Generated = true
	processor.Files = true

	current, err := getResults("")
	if err != nil {
		log.Fatal(err)

		return
	}

	currentGrouped := make(map[string]resultRow)
	for _, v := range current {
		currentGrouped[v.Language] = currentGrouped[v.Language].add(v)
	}

	do := func(workPath string) {
		processor.DirFilePaths = []string{workPath}

		base, err := getResults(workPath)
		if err != nil {
			log.Fatal(err)

			return
		}

		for _, v := range base {
			currentGrouped[v.Language] = currentGrouped[v.Language].addBase(v)
		}
	}

	if f.baseDir == "" {
		err = runAtGitRef(nil, "git", "", f.baseRef, do)
		if err != nil {
			println(err.Error())

			return
		}
	} else {
		do(f.baseDir)
	}

	printTable(currentGrouped, f.all)
}

func printTable(currentGrouped map[string]resultRow, all bool) {
	data := make([][]string, 0, len(currentGrouped))

	for _, v := range currentGrouped {
		hasDiff := false

		row := []string{
			v.Language,
			format(v.Files, v.FilesBase, &hasDiff),
			format(v.Lines, v.LinesBase, &hasDiff),
			format(v.Code, v.CodeBase, &hasDiff),
			format(v.Comments, v.CommentsBase, &hasDiff),
			format(v.Blanks, v.BlanksBase, &hasDiff),
			format(v.Complexity, v.ComplexityBase, &hasDiff),
			formatBytes(v.Bytes, v.BytesBase, &hasDiff),
		}

		if !hasDiff && !all {
			continue
		}

		data = append(data, row)
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i][0] < data[j][0]
	})

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoFormatHeaders(false)
	table.SetHeader([]string{"Language", "Files", "Lines", "Code", "Comments", "Blanks", "Complexity", "Bytes"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.AppendBulk(data) // Add Bulk Data
	table.Render()
}

func formatBytes(current, base int, hasDiff *bool) string {
	if base > current {
		*hasDiff = true

		return fmt.Sprintf("%s (-%s)", bytefmt.ByteSize(uint64(current)), bytefmt.ByteSize(uint64(base-current)))
	} else if base < current {
		*hasDiff = true

		return fmt.Sprintf("%s (+%s)", bytefmt.ByteSize(uint64(current)), bytefmt.ByteSize(uint64(current-base)))
	}

	return bytefmt.ByteSize(uint64(current))
}

func format(current, base int, hasDiff *bool) string {
	if base > current {
		*hasDiff = true

		return fmt.Sprintf("%d (-%d)", current, base-current)
	} else if base < current {
		*hasDiff = true

		return fmt.Sprintf("%d (+%d)", current, current-base)
	}

	return fmt.Sprintf("%d", current)
}

type resultRow struct {
	Language string
	Location string
	Filename string

	Files      int
	Lines      int
	Code       int
	Comments   int
	Blanks     int
	Complexity int
	Bytes      int

	FilesBase      int
	LinesBase      int
	CodeBase       int
	CommentsBase   int
	BlanksBase     int
	ComplexityBase int
	BytesBase      int
}

func (gv resultRow) add(v resultRow) resultRow {
	gv.Language = v.Language
	gv.Files++
	gv.Lines += v.Lines
	gv.Code += v.Code
	gv.Comments += v.Comments
	gv.Blanks += v.Blanks
	gv.Complexity += v.Complexity
	gv.Bytes += v.Bytes

	return gv
}

func (gv resultRow) addBase(v resultRow) resultRow {
	gv.Language = v.Language
	gv.FilesBase++
	gv.LinesBase += v.Lines
	gv.CodeBase += v.Code
	gv.CommentsBase += v.Comments
	gv.BlanksBase += v.Blanks
	gv.ComplexityBase += v.Complexity
	gv.BytesBase += v.Bytes

	return gv
}

func getResults(stripPath string) (map[string]resultRow, error) {
	out, err := processOutput()
	if err != nil {
		return nil, err
	}

	cr := csv.NewReader(bytes.NewReader(out))

	s, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}

	result := make(map[string]resultRow, len(s))

	for i := 1; i < len(s); i++ {
		c := s[i]

		row := resultRow{
			Language:   c[0],
			Location:   c[1],
			Filename:   c[2],
			Lines:      mustParseInt(c[3]),
			Code:       mustParseInt(c[4]),
			Comments:   mustParseInt(c[5]),
			Blanks:     mustParseInt(c[6]),
			Complexity: mustParseInt(c[7]),
			Bytes:      mustParseInt(c[8]),
		}

		row.Location = strings.TrimPrefix(row.Location, stripPath)
		if strings.HasSuffix(row.Filename, "_test.go") {
			row.Language = "Go (test)"
		}

		result[row.Location] = row
	}

	return result, nil
}

func mustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}

	return i
}

func processOutput() ([]byte, error) {
	old := os.Stdout // keep backup of the real stdout

	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	os.Stdout = w

	defer func() { os.Stdout = old }()

	buf := bytes.NewBuffer(nil)
	done := make(chan struct{})

	var copyErr error

	go func() {
		_, copyErr = io.Copy(buf, r)

		close(done)
	}()

	processor.Process()

	if err = w.Close(); err != nil {
		return nil, err
	}

	<-done

	if copyErr != nil {
		return nil, copyErr
	}

	return buf.Bytes(), nil
}
