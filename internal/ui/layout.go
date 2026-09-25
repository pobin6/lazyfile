package ui

type Layout struct {
	LeftWidth   int
	MiddleWidth int
	RightWidth  int
}

func calculateLayout(totalWidth int) Layout {
	left := totalWidth * 20 / 100
	middle := totalWidth * 40 / 100
	return Layout{
		LeftWidth:   left,
		MiddleWidth: middle,
		RightWidth:  totalWidth - left - middle,
	}
}
