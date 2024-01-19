package repton

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"sync"
)

// DirectoryProcessor encapsulates a set of functions to process all or
// some of the files in a directory.
type DirectoryProcessor interface {
	// ProcessFile is called for each file. It's called in batches, each
	// invocation in a goroutine running concurrently with others in the
	// same batch. fileName is a full pathname.
	ProcessFile(fileName string)

	// MinimumFilesNeededForCompletion allows the number of files in each
	// batch to be optimised. For example, extractatlases needs to process
	// at least 6 files to start with, to complete each colour, but after
	// that it may only need to complete a couple more, and should be able
	// to achieve completion without needing to load the entire set.
	MinimumFilesNeededForCompletion() int

	// Finish is called after MinimumFilesNeededForCompletion returns 0.
	Finish()

	// StartBatch is called just before the beginning of each batch
	StartBatch()

	// FinishBatch is called just after the end of each batch
	FinishBatch()
}

// LoadImage loads a PNG, which a DirectoryProcessor will typically find
// useful.
func LoadImage(fileName string) (img image.Image, err error) {
	var fd *os.File
	fd, err = os.Open(fileName)
	if err != nil {
		err = fmt.Errorf("unable to open '%s': %v", fileName, err)
		return
	}
	defer fd.Close()
	img, _, err = image.Decode(fd)
	if err != nil {
		err = fmt.Errorf("unable to decode '%s': %v", fileName, err)
	}
	return
}

// ProcessDirectory uses a DirectoryProcessor to process the files matched
// by globPattern.
func ProcessDirectory(
	globPattern string,
	directoryProcessor DirectoryProcessor,
	maxThreads int,
) error {
	files, err := filepath.Glob(globPattern)
	if err != nil {
		return fmt.Errorf("Error globbing '%s': %v", globPattern, err)
	}
	fileIndex := 0
	finished := len(files) == fileIndex
	if finished {
		fmt.Printf("No files matched pattern '%s'\n", globPattern)
		return nil
	}
	defer directoryProcessor.Finish()
	for !finished {
		numThreads := directoryProcessor.MinimumFilesNeededForCompletion()
		fmt.Printf(
			"DirectoryProcessor needs to process at least %d more files\n",
			numThreads)
		if numThreads == 0 {
			finished = true
			break
		}
		numRemaining := len(files) - fileIndex
		numThreads = min(numThreads, maxThreads, numRemaining)
		fmt.Printf("DirectoryProcessor has %d files remaining, "+
			"starting %d threads\n", numRemaining, numThreads)
		wg := &sync.WaitGroup{}
		directoryProcessor.StartBatch()
		for i := 0; i < numThreads; i++ {
			wg.Add(1)
			go func(fileName string) {
				//fmt.Printf("DirectoryProcessor processing %s\n", fileName)
				directoryProcessor.ProcessFile(fileName)
				wg.Done()
			}(files[fileIndex])
			fileIndex++
		}
		wg.Wait()
		directoryProcessor.FinishBatch()
		finished = len(files) == fileIndex
		fmt.Printf("DirectoryProcessor finished batch of %d, "+
			"%d files remaining\n", numThreads, len(files)-fileIndex)
	}
	return nil
}
