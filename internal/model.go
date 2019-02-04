package jacoco

import "encoding/xml"

type Report struct {
	XMLName     xml.Name      `xml:"report"`
	Name        string        `xml:"name,attr"`
	SessionInfo []SessionInfo `xml:"sessioninfo"`
	Packages    []Package     `xml:"package"`
	Groups      []Group       `xml:"group"`
	Counters    []Counter     `xml:"counter"`
}

type Counter struct {
	Type    string `xml:"type,attr"`
	Missed  int    `xml:"missed,attr"`
	Covered int    `xml:"covered,attr"`
}

type SessionInfo struct {
	ID    string `xml:"id,attr"`
	Start int    `xml:"start,attr"`
	Dump  int    `xml:"dump,attr"`
}

type Line struct {
	Nr int `xml:"nr,attr"`
	Mi int `xml:"mi,attr"`
	Ci int `xml:"ci,attr"`
	Mb int `xml:"mb,attr"`
	Cb int `xml:"cb,attr"`
}

type SourceFile struct {
	Name     string    `xml:"name,attr"`
	Lines    []Line    `xml:"line"`
	Counters []Counter `xml:"counter"`
}

type Method struct {
	Name     string    `xml:"name,attr"`
	Desc     string    `xml:"desc,attr"`
	Line     int       `xml:"line,attr"`
	Counters []Counter `xml:"counter"`
}

type Class struct {
	Name           string    `xml:"name,attr"`
	Sourcefilename string    `xml:"sourcefilename,attr"`
	Methods        []Method  `xml:"method"`
	Counters       []Counter `xml:"counter"`
}

type Package struct {
	Name        string       `xml:"name,attr"`
	Classes     []Class      `xml:"class"`
	SourceFiles []SourceFile `xml:"sourcefile"`
	Counters    []Counter    `xml:"counter"`
}

type Group struct {
	Name     string    `xml:"name,attr"`
	Packages []Package `xml:"package"`
	Groups   []Group   `xml:"group"`
	Counters []Counter `xml:"counter"`
}
