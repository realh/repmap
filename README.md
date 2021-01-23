repmap
======

This is a set of tools for analysing screenshots from Repton 2 Editor (Windows
PC version) and generating concise, machine-readable representations of the map
data. It can be used as an alternative to [Gerald Holdsworth's Repton Map
Decoder](http://www.reptonresourcepage.co.uk/ReptonMapDisplay.php).

To build the tools you'll need a working go (aka golang) toolchain. On Linux
and Mac OS running `make all` should work, but this is untested. On Windows
you'll probably need to edit Makefile to add .exe extensions to each target
(and install MinGW or similar). In case the Makefile doesn't work, you can
build each tool individually by running `go build -v cmd/img2map/img2map.go`,
and similarly for the other tools by substituting their names for `img2map`.

The tools take advantage of go's concurrency, so their logging output on stderr
may appear in an unexpected order.

img2map
-------
This is the main tool. It loads one or more screenshots of Repton 2 Editor's
main window and outputs corresponding text files representing the map data in a
more manageable format. The first line of each output file is the colour scheme
("Red", "Blue" etc). Each following line represents a row of the map with each
character representing a tile. The tile types are numbered 0-33 in the order
shown in `pkg/repton2/tiles.go`. This ordering is based on the order in which
the tiles appear in the 6x6 selecter part of the editor, and differs slightly
from the ordering that Repton Map Decoder uses. These values are mapped to
characters as follows:

```
0 = .
1-9 = 1-9
10-33 = A-X
```

To run it:

`./img2map input reftilehashes.json output`

where `input` can be one screenshot, a scenario folder containing a set of
screenshots ("01.png" - "20.png"), or a folder containing several scenario
folders. `output` is the output, which will either be a single text file, a
folder containing a scenario's worth ("01.txt" - "20.txt"), or several scenario
folders, mirroring the input. `reftilehashes.json` is supplied with this
repository and holds hash values for all the different tile sprites which
img2map uses to work out tile types from pixel data.

The output on stderr includes the number of puzzle pieces found, which can
serve as a useful warning that matching may have gone wrong. The default number
of puzzle pieces per scenario is 104.

refhash
-------
This is the tool used to generate `reftilehashes.json`, so you shouldn't need
it. In case you do, run it with:

`./refhash input_folder > reftilehashes.json`

`input_folder` must contain a set of editor screenshots named after Repton's
colour themes ("Blue.png", "Cyan.png", "Green.png", "Magenta.png", "Red.png",
"Orange.png"). Each must show a special dummy map constructed as:

```
WX......................
.12345..................
6789AB..................
CDEFGH..................
IJKLMN..................
OPQRST..................
.V......................
........................
```

where the characters correspond to the table above. These files are not
supplied here. The output is on stdout, hence `>`.

asc2csv, csv2asc
----------------
These two utilities convert between repmap's ASCII format and the CSV-based
format from Repton Map Decoder. The input argument is a file, and the output
is on stdout. The input argument can be omitted to use stdin. Examples of usage:

```
./asc2csv levels/Jungle/01.txt > levels/Jungle/01.csv
./csv2asc levels/Jungle/01.csv > levels/Jungle/01.txt
```

This can be useful for comparing the outputs of img2map and Repton Map Decoder,
using something like UNIX diff. An option like `--ignore-all-space` may help in
case you're comparing files with UNIX vs Windows line endings.

Licence
-------
ISC Licence (ISC)
Copyright 2021 Tony Houghton <h@realh.co.uk>

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH
REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY AND
FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT,
INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR
OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
PERFORMANCE OF THIS SOFTWARE.
