module github.com/encrypto/encrypto

go 1.25.0

require (
	github.com/filemax/filemax/pkg/termui v0.0.0
	golang.org/x/crypto v0.48.0
)

require (
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/term v0.40.0 // indirect
	howett.net/plist v1.0.1 // indirect
)

replace github.com/filemax/filemax/pkg/termui => ../pkg/termui
