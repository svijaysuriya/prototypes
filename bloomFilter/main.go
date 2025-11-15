package main

type bloomFilter struct {
	nHashFunction int
	size          int
	bitArray      []bool
}

func newBloomFilter(size, nHashFunction int) bloomFilter {
	return bloomFilter{
		nHashFunction: nHashFunction,
		size:          size,
		bitArray:      make([]bool, size),
	}
}
func hash(item string, i int) int {
	sum := 0
	for j := 0; j < len(item); j++ {
		sum += int(item[j])
	}
	return sum + i
}
func (bf *bloomFilter) add(item string) {
	for i := 0; i < bf.nHashFunction; i++ {
		hash := hash(item, i)
		bf.bitArray[hash%bf.size] = true
	}
}
func (bf *bloomFilter) contains(item string) bool {
	for i := 0; i < bf.nHashFunction; i++ {
		hash := hash(item, i)
		if !bf.bitArray[hash%bf.size] {
			return false
		}
	}
	return true
}

func main() {
	bf := newBloomFilter(100, 3)
	bf.add("hello")
	// fmt.Println(bf.bitArray)
	bf.add("vijay")
	// fmt.Println(bf.bitArray)
	bf.add("suriya")
	// fmt.Println(bf.bitArray)
	bf.add("kabilan")
	// fmt.Println(bf.bitArray)
	bf.add("vimal")
	// fmt.Println(bf.bitArray)
	// bf.add("vimalan")
	// fmt.Println(bf.bitArray)
	elementsInBloomFilter := []string{"hello", "vijay", "suriya", "kabilan", "vimal"}
	for _, element := range elementsInBloomFilter {
		if bf.contains(element) {
			println(element, " is in bloom filter", "\t[True Positive]")
		} else {
			println(element, "False Negative - case not possible in BF is not in bloom filter")
		}
	}
	elementsNotInBloomFilter := []string{"vimalan", "saraseswari", "dhina", "sun", "apple", "ballss"}
	for _, element := range elementsNotInBloomFilter {
		if bf.contains(element) {
			println("!!!False Positive [type1 error]=> ", element, " - is in bloom filter")
		} else {
			println(element, " is not in bloom filter", "\t[True Negative]")
		}
	}
}
