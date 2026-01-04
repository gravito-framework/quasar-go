package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

func main() {
	procs, err := process.Processes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing processes: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Total processes: %d\n", len(procs))
	
	count := 0
	for _, p := range procs {
		cmdline, err := p.Cmdline()
		if err != nil {
			continue
		}
		
		if strings.Contains(cmdline, "artisan") {
			fmt.Printf("Found artisan: %s\n", cmdline)
			count++
		}
	}
	fmt.Printf("Matches found: %d\n", count)
}
