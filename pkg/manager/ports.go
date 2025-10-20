package manager

import (
	"fmt"
	"math/bits"
	"sync"
)

// portAllocator provides efficient port allocation using a bitmap for O(1) operations.
// The bitmap approach prevents unbounded memory growth and simplifies port management.
type portAllocator struct {
	mu sync.Mutex

	// Bitmap for O(1) allocation/release
	// Each bit represents a port (1 = allocated, 0 = free)
	bitmap []uint64 // Each uint64 covers 64 ports

	// Map port to instance name for cleanup operations
	allocated map[int]string

	minPort   int
	maxPort   int
	rangeSize int
}

// NewPortAllocator creates a new port allocator for the given port range.
// Returns an error if the port range is invalid.
func NewPortAllocator(minPort, maxPort int) (*portAllocator, error) {
	if minPort <= 0 || maxPort <= 0 {
		return nil, fmt.Errorf("invalid port range: min=%d, max=%d (must be > 0)", minPort, maxPort)
	}
	if minPort > maxPort {
		return nil, fmt.Errorf("invalid port range: min=%d > max=%d", minPort, maxPort)
	}

	rangeSize := maxPort - minPort + 1
	bitmapSize := (rangeSize + 63) / 64 // Round up to nearest uint64

	return &portAllocator{
		bitmap:    make([]uint64, bitmapSize),
		allocated: make(map[int]string),
		minPort:   minPort,
		maxPort:   maxPort,
		rangeSize: rangeSize,
	}, nil
}

// allocate finds and allocates the first available port for the given instance.
// Returns the allocated port or an error if no ports are available.
func (p *portAllocator) allocate(instanceName string) (int, error) {
	if instanceName == "" {
		return 0, fmt.Errorf("instance name cannot be empty")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	port, err := p.findFirstFreeBit()
	if err != nil {
		return 0, err
	}

	p.setBit(port)
	p.allocated[port] = instanceName

	return port, nil
}

// allocateSpecific allocates a specific port for the given instance.
// Returns an error if the port is already allocated or out of range.
func (p *portAllocator) allocateSpecific(port int, instanceName string) error {
	if instanceName == "" {
		return fmt.Errorf("instance name cannot be empty")
	}
	if port < p.minPort || port > p.maxPort {
		return fmt.Errorf("port %d is out of range [%d-%d]", port, p.minPort, p.maxPort)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isBitSet(port) {
		return fmt.Errorf("port %d is already allocated", port)
	}

	p.setBit(port)
	p.allocated[port] = instanceName

	return nil
}

// release releases a specific port, making it available for reuse.
// Returns an error if the port is not allocated.
func (p *portAllocator) release(port int) error {
	if port < p.minPort || port > p.maxPort {
		return fmt.Errorf("port %d is out of range [%d-%d]", port, p.minPort, p.maxPort)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isBitSet(port) {
		return fmt.Errorf("port %d is not allocated", port)
	}

	p.clearBit(port)
	delete(p.allocated, port)

	return nil
}

// releaseByInstance releases all ports allocated to the given instance.
// This is useful for cleanup when deleting or updating an instance.
// Returns the number of ports released.
func (p *portAllocator) releaseByInstance(instanceName string) int {
	if instanceName == "" {
		return 0
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	portsToRelease := make([]int, 0)
	for port, name := range p.allocated {
		if name == instanceName {
			portsToRelease = append(portsToRelease, port)
		}
	}

	for _, port := range portsToRelease {
		p.clearBit(port)
		delete(p.allocated, port)
	}

	return len(portsToRelease)
}

// --- Internal bitmap operations ---

// portToBitPos converts a port number to bitmap array index and bit position.
func (p *portAllocator) portToBitPos(port int) (index int, bit uint) {
	offset := port - p.minPort
	index = offset / 64
	bit = uint(offset % 64)
	return
}

// setBit marks a port as allocated in the bitmap.
func (p *portAllocator) setBit(port int) {
	index, bit := p.portToBitPos(port)
	p.bitmap[index] |= (1 << bit)
}

// clearBit marks a port as free in the bitmap.
func (p *portAllocator) clearBit(port int) {
	index, bit := p.portToBitPos(port)
	p.bitmap[index] &^= (1 << bit)
}

// isBitSet checks if a port is allocated in the bitmap.
func (p *portAllocator) isBitSet(port int) bool {
	index, bit := p.portToBitPos(port)
	return (p.bitmap[index] & (1 << bit)) != 0
}

// findFirstFreeBit scans the bitmap to find the first unallocated port.
// Returns the port number or an error if no ports are available.
func (p *portAllocator) findFirstFreeBit() (int, error) {
	for i, word := range p.bitmap {
		if word != ^uint64(0) { // Not all bits are set (some ports are free)
			// Find the first 0 bit in this word
			// XOR with all 1s to flip bits, then find first 1 (which was 0)
			bit := bits.TrailingZeros64(^word)
			port := p.minPort + (i * 64) + bit

			// Ensure we don't go beyond maxPort due to bitmap rounding
			if port <= p.maxPort {
				return port, nil
			}
		}
	}

	return 0, fmt.Errorf("no available ports in range [%d-%d]", p.minPort, p.maxPort)
}
