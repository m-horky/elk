// Package fs provides filesystem capabilities.
//
// Use FS in consumers and inject an implementation rather than calling
// the operating system directly. Tests can provide a small fake that
// implements the same interface.
//
// Read opens and validates the file before returning at most the requested
// number of its bytes. Reads that exceed the limit return the bounded bytes
// and ErrTruncated:
//
//	func loadConfig(filesystem fs.FS, path string) ([]byte, error) {
//	     return filesystem.Read(path, 1024*1024)
//	}
//
// The File returned by Open is a descriptor-backed read handle and must be
// closed by the caller.
package fs
