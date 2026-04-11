### Binary search: 
compressToMaxSize now finds the best quality in ≤7 encodes instead of up to 9, and picks the highest quality that still fits under the size cap (the old linear scan would stop at the first fit, potentially using a lower quality than needed).

### Worker pool: 
main now spawns up to runtime.NumCPU() goroutines in parallel, so all cores are used instead of one.

reduce 166.63s -> 63.21s