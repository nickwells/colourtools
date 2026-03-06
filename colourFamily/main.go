package main

import "os"

func main() {
	prog := newProg()
	ps := makeParamSet(prog)
	ps.Parse()

	prog.run()
	os.Exit(prog.exitStatus)
}
