package main

import (
	"fmt"
	"os"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
)

// benchLength defines the number of turns for the Game of Life simulation in benchmarks
const benchLength = 1000

// BenchmarkGol is a benchmark test function for the Game of Life simulation.
// It varies the number of worker threads to assess performance.
func BenchmarkGol(b *testing.B) {

  // Iterate over threads and disable standard output to ensure only benchmark results are displayed
  for threads := 1; threads <= 16; threads++ {
		os.Stdout = nil // Disable all program output apart from benchmark results
		
    p := gol.Params{
			Turns:       benchLength,
			Threads:     threads,
			ImageWidth:  512,
			ImageHeight: 512,
		}

    // Construct a unique name for each benchmark configuration
		name := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
		
    // Run a sub-benchmark with the specified configuration name.
    b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
        // Create a channel to communicate with the GoL simulation.
				events := make(chan gol.Event)
        // Start the Game of Life simulation in a separate goroutine.
				go gol.Run(p, events, nil)
				// Consume and discard events from the simulation until it ends.
				for range events {

				}
			}
		})
	}
}
