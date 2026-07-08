package domain

import "time"

type ProfilePackage struct {
	System     string
	DrvPath    string
	StorePath  string
	OutputName string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Package    Package
}

type Package struct {
	Attr        string
	Name        string
	Description string
	Version     string
	License     License
	Outputs     []string
	Platforms   []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type License struct {
	Open bool
	Name string
}
