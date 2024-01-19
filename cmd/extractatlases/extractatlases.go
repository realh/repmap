package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/realh/repmap/pkg/repton"
)

// extractatlases scans a set of images in a directory. These images should
// be full-size screenshots from Repton Resource Pages map, typically a
// scenario's worth. There need to be enough so that between them they contain
// every possible sprite that may appear in such a map view in every colour
// scheme. The output is a directory containing an atlas for each colour. The
// sprites in each atlas will (hopefully) always be in the same order, but
// that order is undefined.

type AtlasExtractor struct {
	NumColoursFound  int
	FoundColour      [6]bool
	ColourChannel    chan int
	BatchDoneChannel chan bool
	Wg               *sync.WaitGroup
}

func (ae *AtlasExtractor) ProcessFile(fileName string) {
	img, error := repton.LoadImage(fileName)
	if error != nil {
		fmt.Println(error)
		return
	}
	ae.Wg.Add(1)
	desc := filepath.Base(fileName)
	dominantColour := repton.DetectThemeOfEntireImage(img, desc)
	leaf := filepath.Base(fileName)
	if dominantColour != -1 {
		fmt.Printf("Finished processing %s with colour %s\n",
			leaf, repton.ColourNames[dominantColour])
		ae.ColourChannel <- dominantColour
	} else {
		fmt.Printf("Finished processing %s with no dominant colour\n", leaf)
	}
	ae.Wg.Done()
}

func (ae *AtlasExtractor) MinimumFilesNeededForCompletion() int {
	fmt.Printf("Found %d colours so far\n", ae.NumColoursFound)
	return max(1, 6-ae.NumColoursFound)
}

func (ae *AtlasExtractor) Finish() {

}

func (ae *AtlasExtractor) Start(directory string) {
	repton.ProcessDirectory(directory+"/*.png", ae, 6)
}

func (ae *AtlasExtractor) StartBatch() {
	ae.Wg = &sync.WaitGroup{}
	ae.Wg.Add(1)
	// Each time a dominant colour is detected in a file it is sent down
	// this channel to the goroutine below to avoid race conditions when
	// detecting whether that colour was already detected.
	ae.ColourChannel = make(chan int)
	ae.BatchDoneChannel = make(chan bool)
	// This is the "colour detection" goroutine; when it receives -1 it
	// closes BatchDoneChannel and exits
	go func() {
		for running := true; running; {
			colour := <-ae.ColourChannel
			if colour == -1 {
				fmt.Println("Colour detector got -1, exiting")
				running = false
				break
			}
			fmt.Printf("Found %s (previously found %v), total %d\n",
				repton.ColourNames[colour],
				ae.FoundColour[colour],
				ae.NumColoursFound+1,
			)
			if !ae.FoundColour[colour] {
				ae.FoundColour[colour] = true
				ae.NumColoursFound++
			}
		}
		close(ae.BatchDoneChannel)
	}()
}

func (ae *AtlasExtractor) FinishBatch() {
	fmt.Println("Finished batch, waiting for image processors")
	ae.Wg.Done()
	ae.Wg.Wait()
	fmt.Println("Shutting down and awaiting colour detection goroutine")
	ae.ColourChannel <- -1
	<-ae.BatchDoneChannel
	close(ae.ColourChannel)
	ae.ColourChannel = nil
	ae.BatchDoneChannel = nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("extractatlases takes 2 arguments: ")
		fmt.Println("input folder, output folder")
		os.Exit(1)
	}
	ae := AtlasExtractor{}
	ae.Start(os.Args[1])
}
