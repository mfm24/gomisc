package main

import "fmt"
import "os"
import "image"
import "image/png"
import "image/color"
import "io"
import "net/http"
import "log"
import "strings"
import "strconv"
import "runtime"

type mand_desc struct{
	w, h int
	max_it int
	sx, ex, sy, ey float64	
}

type mand_request struct{
	mand_desc
	out io.Writer
	donechan chan int
}

func def_palette() color.Palette {
	ret := make(color.Palette, 256)
	ret[0] = color.RGBA{uint8(0),uint8(0),uint8(0), uint8(255)}
	for i := 1; i < 256; i++ {
		ret[i] = color.YCbCr{uint8(128), uint8((i*191) % 256), uint8(((i+1)*191)%256)}
	}
	return ret
}
//returns a mandelbrot with the dimensions w,h and for the rectangle
// sx,sy, ex, ey. Returns an image.Paletted
func make_mand(w,h,max_it int, sx, ex, sy, ey float64) *image.Paletted{
	ret := image.NewPaletted(image.Rect(0,0,w,h), def_palette())
	xpos, ypos := sx, sy
	dx, dy := (ex-sx)/float64(w), (ey-sy)/float64(h)
	for i := 0; i < h; i++ { 
		xpos = sx
		for j := 0; j < w; j++ {
			itcount := 0
			for re, im := xpos, ypos; itcount < max_it && (re*re+im*im)<4.0; itcount++ {
				re, im = xpos + re*re - im*im, ypos + 2*im*re
				itcount += 1
			}
			if itcount==max_it{
				ret.SetColorIndex(j,i,0)
			} else {
				ret.SetColorIndex(j, i, uint8(1 + itcount%255))
			}
			
			xpos += dx
		}
		ypos += dy
	}
	return ret
}

func mand_handler(inchannel chan mand_request) {
	for m := range(inchannel) {
		ret, ok := cache[m.mand_desc]
		if !ok{
			ret = make_mand(m.w, m.h, m.max_it, m.sx, m.ex, m.sy, m.ey)
			cache[m.mand_desc] = ret
		} else {
			fmt.Println("Hitting cache!")
		}
		png.Encode(m.out, ret)
		m.donechan <- 1
	}
}

func write_single() {
    w, h := 512, 512
    mr := make(chan mand_request)
    go mand_handler(mr)

    f,_ := os.Create("out.png")
    defer f.Close()
    
    m := mand_request{ mand_desc{w,h,32, -2,2,-2,2}, f, make(chan int)}
    mr <- m
    <- m.donechan

    fmt.Println("Done!")
}

var mand_request_channel = make(chan mand_request)
var cache = map[mand_desc] *image.Paletted {}
 
func mand_server(wr http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path, "/")
	// since path is always /... parts[0] is ''
	//we use sy,sx,ey,ex,max_its
	sy,err := strconv.ParseFloat(parts[1], 64)
	sx,err := strconv.ParseFloat(parts[2], 64)
	ey,err := strconv.ParseFloat(parts[3], 64)
	ex,err := strconv.ParseFloat(parts[4], 64)
	max_it, err := strconv.ParseInt(parts[5], 0, 0)
	_ = err
	fmt.Println("Requesting", req.URL.Path, sx, sy, ex, ey, max_it)
	m := mand_request{ 
		mand_desc{256, 256, int(max_it), sx,ex,sy,ey}, 
		wr, make(chan int)}
	mand_request_channel <- m
    <- m.donechan
}

func serve_pngs() {
	fmt.Println("Setting Max Procs to", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		go mand_handler(mand_request_channel)
	}
    
    fmt.Println("Starting server...")
	http.HandleFunc("/", mand_server)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func main() {
	//write_single()
	serve_pngs()
}