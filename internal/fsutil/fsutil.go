package fsutil

import "os"

func EnsureDir(dir string) error {
	// 0755 → owner: rwx, group: rx, others: rx
	// r = 4, w = 2, x = 1, r = list, w = create/delete, x = traverse
	return os.MkdirAll(dir, 0755)
}
