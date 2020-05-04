// Surface1 computes an SVG rendering of a 3-D surface function.
package main

import (
	"fmt"
	"image/color"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
)

const (
	width, height = 600, 320            // canvas size in pixels
	cells         = 100                 // number of grid cells
	xyrange       = 30.0                // axis ranges (-xyrange..+xyrange)
	xyscale       = width / 2 / xyrange // pixels per x or y unit
	zscale        = height * 0.4        // pixels per z unit
	angle         = math.Pi / 6         // angle of x, y axes (=30°)
)

var (
	sin30, cos30 = math.Sin(angle), math.Cos(angle) // sin(30°), cos(30°)
	red, blue    = color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}
)

type prefs struct {
	width, height   int
	cells           int
	xyrange         float64
	_xyscale        float64
	_zscale         float64
	lowest, highest color.RGBA
	f               func(float64, float64) float64
}

func main() {
	// http.Handle("/", http.FileServer(http.Dir("./front")))
	// FS() is created by esc and returns a http.Filesystem
	http.Handle("/", http.FileServer(FS(false)))
	http.HandleFunc("/draw", handler)
	fmt.Println("Starting server at http://localhost:80, enjoy!")
	fmt.Println("Press Ctrl+C for shut down")
	log.Fatal(http.ListenAndServe("localhost:80", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p, err := getPrefs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	draw(w, p)
}

func getPrefs(r *http.Request) (p *prefs, err error) {
	p = &prefs{}
	p.width, p.height, p.cells = width, height, cells
	p.xyrange, p._xyscale, p._zscale = xyrange, xyscale, zscale
	p.lowest, p.highest = blue, red
	p.f = f1

	for k, v := range r.Form {
		if len(v) == 0 {
			err = fmt.Errorf("no value for parameter %q", k)
			return
		}
		val := v[0]
		switch k {
		case "width":
			p.width, err = strconv.Atoi(val)
			if err != nil {
				err = fmt.Errorf("parameter %q value should be an integer", k)
				return
			}
		case "height":
			p.height, err = strconv.Atoi(val)
			if err != nil {
				err = fmt.Errorf("parameter %q value should be an integer", k)
				return
			}
		case "cells":
			p.cells, err = strconv.Atoi(val)
			if err != nil {
				err = fmt.Errorf("parameter %q value should be an integer", k)
				return
			}
		case "xyrange":
			p.xyrange, err = strconv.ParseFloat(val, 64)
			if err != nil {
				err = fmt.Errorf("parameter %q value should be a float", k)
				return
			}

		case "lowest":
			lc, errc := HEXtoRGBA(val)
			if errc != nil {
				err = fmt.Errorf("parameter %q value error: %v", k, errc)
				return
			}
			p.lowest = *lc
		case "highest":
			hc, errc := HEXtoRGBA(val)
			if errc != nil {
				err = fmt.Errorf("parameter %q value error: %v", k, errc)
				return
			}
			p.highest = *hc
		case "func":
			switch val {
			case "f1", "schaffer":
				p.f = f1
			case "f2", "eggbox":
				p.f = f2
			case "f3", "sinc":
				p.f = f3
			case "f4", "moguls":
				p.f = f4
			case "f5", "saddle":
				p.f = f5
			default:
				err = fmt.Errorf("parameter %q value should be one of %q, %q, %q, %q, %q", k, "schaffer", "eggbox", "sinc", "moguls", "saddle")
			}
		case "background": // it is fine, just skip it
		default:
			err = fmt.Errorf("wrong parameter %q", k)
			return
		}
	}

	p._xyscale = float64(p.width) / 2 / p.xyrange
	p._zscale = float64(p.height) * 0.4

	return
}

func draw(w io.Writer, p *prefs) {
	fmt.Fprintf(w, "<svg xmlns='http://www.w3.org/2000/svg' "+
		"style='stroke: none; fill: none; stroke-width: 0.7' "+
		"width='%d' height='%d'>", p.width, p.height)
	for i := 0; i < p.cells; i++ {
		for j := 0; j < p.cells; j++ {
			ax, ay, z1, ok1 := corner(i+1, j, p)
			bx, by, z2, ok2 := corner(i, j, p)
			cx, cy, z3, ok3 := corner(i, j+1, p)
			dx, dy, z4, ok4 := corner(i+1, j+1, p)
			if !ok1 || !ok2 || !ok3 || !ok4 {
				continue
			}
			filler := gradient(p.lowest, p.highest, prop(avg(z1, z2, z3, z4)))
			fmt.Fprintf(w, "<polygon points='%g,%g %g,%g %g,%g %g,%g' fill='%s'/>\n", ax, ay, bx, by, cx, cy, dx, dy, RGBAtoHEX(filler))
		}
	}
	fmt.Fprintf(w, "</svg>")
}

func avg(nums ...float64) float64 {
	var total float64
	for _, f := range nums {
		total += f
	}
	return total / float64(len(nums))
}

// RGBAtoHEX converts RGBA color to HEX
func RGBAtoHEX(c color.RGBA) string {
	return fmt.Sprintf("#%.2x%.2x%.2x", c.R, c.G, c.B)
}

// HEXtoRGBA converts HEX color to RGBA
func HEXtoRGBA(hex string) (*color.RGBA, error) {
	switch len(hex) {
	case 4, 7: // it is fine
	default:
		return nil, fmt.Errorf("bad hex %q", hex)
	}

	// cut # symbol
	hex = hex[1:]

	// short notation handling
	if len(hex) == 3 {
		hex += hex
	}

	values, err := strconv.ParseUint(string(hex), 16, 32)
	if err != nil {
		return nil, fmt.Errorf("bad hex %q: %v", hex, err)
	}

	return &color.RGBA{
		R: uint8(values >> 16),
		G: uint8((values >> 8) & 0xFF),
		B: uint8(values & 0xFF),
		A: 255,
	}, nil
}

func gradient(color1, color2 color.RGBA, percent float64) color.RGBA {
	r := color1.R + byte(percent*float64(color2.R-color1.R))
	g := color1.G + byte(percent*float64(color2.G-color1.G))
	b := color1.B + byte(percent*float64(color2.B-color1.B))
	a := color1.B + byte(percent*float64(color2.A-color1.A))
	return color.RGBA{r, g, b, a}
}

func prop(f float64) float64 {
	return f/2 + 0.5
}

func corner(i, j int, p *prefs) (float64, float64, float64, bool) {
	// Find point (x,y) at corner of cell (i,j).
	x := p.xyrange * (float64(i)/float64(p.cells) - 0.5)
	y := p.xyrange * (float64(j)/float64(p.cells) - 0.5)

	// Compute surface height z.
	z := p.f(x, y)
	if math.IsInf(z, 0) || math.IsNaN(z) {
		return math.NaN(), math.NaN(), math.NaN(), false
	}

	// Project (x,y,z) isometrically onto 2-D SVG canvas (sx,sy).
	sx := float64(p.width)/2 + (x-y)*cos30*p._xyscale
	sy := float64(p.height)/2 + (x+y)*sin30*p._xyscale - z*p._zscale

	return sx, sy, z, true
}

// schaffer
func f1(x, y float64) float64 {
	r := math.Hypot(x, y) // distance from (0,0)
	return math.Sin(r)    // r
}

// eggbox
func f2(x, y float64) float64 {
	return math.Pow(2, math.Sin(x)) * math.Pow(2, math.Sin(y)) / 12
}

// sinc
func f3(x, y float64) float64 {
	r := math.Hypot(x, y)
	return math.Sin(r) / r
}

// moguls
func f4(x, y float64) float64 {
	return math.Sin(x*y/10) / 10
}

// saddle
func f5(x, y float64) float64 {
	r := x*x - y*y
	return r
}
