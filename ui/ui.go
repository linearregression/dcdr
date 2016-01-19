package ui

import (
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/vsco/dcdr/models"
)

var (
	headerFmt = color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt = color.New(color.FgYellow).SprintfFunc()
)

type UI struct {
	tbl table.Table
}

func New() (u *UI) {
	tbl := table.New("Name", "Type", "Value", "Comment").
		WithHeaderFormatter(headerFmt).
		WithFirstColumnFormatter(columnFmt)

	u = &UI{
		tbl: tbl,
	}

	return
}

func (u *UI) DrawTable(features models.Features) {
	for _, feature := range features {
		u.tbl.AddRow(feature.Name, feature.FeatureType, feature.Value, feature.Comment)
	}

	u.tbl.Print()
}