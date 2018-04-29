package server

import "sort"

// Uint64Slice attaches the methods of Interface to []uint64, sorting in increasing order
type Uint64Slice []uint64

func (p Uint64Slice) Len() int           { return len(p) }
func (p Uint64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Uint64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Sort is a convenience method
func (p Uint64Slice) Sort() { sort.Sort(p) }
