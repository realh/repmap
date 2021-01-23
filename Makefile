.PHONY: all

all: asc2csv csv2asc img2map refhash

asc2csv:
	go build -v cmd/asc2csv/asc2csv.go

csv2asc:
	go build -v cmd/csv2asc/csv2asc.go

img2map:
	go build -v cmd/img2map/img2map.go

refhash:
	go build -v cmd/refhash/refhash.go
