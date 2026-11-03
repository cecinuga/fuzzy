package generator

import (
	"bufio"
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	streamThreshold = 25 << 20 // bytes: a bigger dictionary stays on disk
	blockSize       = 16 << 10 // bytes a block reaches before it ends, at the end of a line
	blockCacheSize  = 16 << 20 // bytes of blocks kept in memory
	indexBufferSize = 1 << 20
)

// wordlist reads a big dictionary by position, so its memory doesn't depend on
// the size of the file. It knows where each block starts and nothing about the
// lines in it: a value is found by reading its block, which the workers share.
// Positions, not a shared cursor, are what keep a seed reproducible: which value
// a job gets can't depend on which worker asked first.
type wordlist struct {
	path    string
	file    *os.File
	offsets []int64 // where each block starts, then where the last one ends
	size    int     // non-blank lines
	cache   blockCache
	failure atomic.Pointer[error]
}

// openWordlist indexes the file with one pass over it.
func openWordlist(path string) (*Dictionary, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read dictionary: %w", err)
	}
	d := &Dictionary{lines: &wordlist{
		path:    path,
		file:    file,
		offsets: []int64{0},
		cache:   blockCache{limit: blockCacheSize, blocks: map[int]*list.Element{}},
	}}
	if err := d.index(); err != nil {
		file.Close()
		return nil, err
	}
	return d, nil
}

func (d *Dictionary) index() error {
	w := d.lines
	reader := bufio.NewReaderSize(w.file, indexBufferSize)
	var offset int64
	blank, whole := true, true // what is known of the current line, which can outgrow the buffer
	for {
		chunk, err := reader.ReadSlice('\n')
		offset += int64(len(chunk))
		blank = blank && len(bytes.TrimSpace(chunk)) == 0
		if err == bufio.ErrBufferFull {
			whole = false
			continue
		}
		if err != nil && err != io.EOF {
			return fmt.Errorf("read dictionary: %w", err)
		}
		if !blank {
			w.size++
		}
		if whole {
			d.addBool(string(bytes.TrimRight(chunk, "\r\n")))
		}
		start := w.offsets[len(w.offsets)-1]
		if offset-start >= blockSize || (err == io.EOF && offset > start) {
			w.offsets = append(w.offsets, offset)
		}
		if err == io.EOF {
			break
		}
		blank, whole = true, true
	}
	if w.size == 0 {
		return errors.New("dictionary is empty: " + w.path)
	}
	return nil
}

// addBool keeps each boolean once, however many lines repeat it.
func (d *Dictionary) addBool(line string) {
	switch line {
	case "true":
		if !slices.Contains(d.Bools, true) {
			d.Bools = append(d.Bools, true)
		}
	case "false":
		if !slices.Contains(d.Bools, false) {
			d.Bools = append(d.Bools, false)
		}
	}
}

func (w *wordlist) err() error {
	if err := w.failure.Load(); err != nil {
		return *err
	}
	return nil
}

func (w *wordlist) fail(err error) { w.failure.CompareAndSwap(nil, &err) }

func (w *wordlist) read(block int) (string, error) {
	buf := make([]byte, w.offsets[block+1]-w.offsets[block])
	if _, err := w.file.ReadAt(buf, w.offsets[block]); err != nil {
		return "", fmt.Errorf("read %s: %w", w.path, err)
	}
	return string(buf), nil
}

// view counts the lines accept takes, one pass over the file with every core.
// It goes around the cache: a pass would only empty it. A block that can't be
// read counts as having none, so nothing asks for its values later.
func (w *wordlist) view(accept func(string) bool, value func(string) any) *view {
	blocks := len(w.offsets) - 1
	counts := make([]int, blocks)
	var next atomic.Int64
	var workers sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), blocks) {
		workers.Go(func() {
			for {
				block := int(next.Add(1)) - 1
				if block >= blocks {
					return
				}
				text, err := w.read(block)
				if err != nil {
					w.fail(err)
					continue
				}
				count := 0
				for raw := range strings.Lines(text) {
					if line, ok := cleanLine(raw); ok && accept(line) {
						count++
					}
				}
				counts[block] = count
			}
		})
	}
	workers.Wait()

	before := make([]int, blocks+1)
	for i, count := range counts {
		before[i+1] = before[i] + count
	}
	return &view{list: w, before: before, accept: accept, value: value}
}

// view is the lines of a wordlist that fit a set of constraints, in file order.
type view struct {
	list   *wordlist
	before []int // fitting lines before each block, then the total
	accept func(string) bool
	value  func(string) any
}

func (v *view) size() int { return v.before[len(v.before)-1] }

// at returns the i-th value, by counting the fitting lines of its block again.
// ok is false if the file can't be read or isn't what was indexed any more.
func (v *view) at(i int) (value any, ok bool) {
	next, _ := slices.BinarySearch(v.before, i+1)
	block := next - 1
	text, err := v.list.cache.get(block, v.list)
	if err != nil {
		v.list.fail(err)
		return nil, false
	}
	rank := i - v.before[block]
	for raw := range strings.Lines(text) {
		line, ok := cleanLine(raw)
		if !ok || !v.accept(line) {
			continue
		}
		if rank == 0 {
			// A copy: a value that lands in a result would otherwise keep its whole block alive.
			return v.value(strings.Clone(line)), true
		}
		rank--
	}
	v.list.fail(fmt.Errorf("%s changed during the run", v.list.path))
	return nil, false
}

// blockCache keeps the blocks used last. Parameters walk consecutive values, so
// a block serves many requests. Workers missing the same block share one read.
type blockCache struct {
	mu     sync.Mutex
	limit  int64
	used   int64
	blocks map[int]*list.Element
	order  list.List // most recently used first
}

type cachedBlock struct {
	index int
	size  int64
	once  sync.Once
	text  string
	err   error
}

func (c *blockCache) get(index int, w *wordlist) (string, error) {
	c.mu.Lock()
	element, ok := c.blocks[index]
	if ok {
		c.order.MoveToFront(element)
	} else {
		element = c.order.PushFront(&cachedBlock{index: index, size: w.offsets[index+1] - w.offsets[index]})
		c.blocks[index] = element
		c.used += element.Value.(*cachedBlock).size
		c.evict()
	}
	c.mu.Unlock()

	block := element.Value.(*cachedBlock)
	block.once.Do(func() { block.text, block.err = w.read(index) })
	return block.text, block.err
}

// evict always leaves the newest block, even one bigger than the limit.
func (c *blockCache) evict() {
	for c.used > c.limit && c.order.Len() > 1 {
		oldest := c.order.Remove(c.order.Back()).(*cachedBlock)
		delete(c.blocks, oldest.index)
		c.used -= oldest.size
	}
}
