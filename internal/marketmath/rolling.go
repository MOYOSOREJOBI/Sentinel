package marketmath

import "math"

type Ring struct {
	buf   []float64
	size  int
	idx   int
	count int
	sum   float64
	sumSq float64
}

func NewRing(size int) *Ring {
	return &Ring{buf: make([]float64, size), size: size}
}

func (r *Ring) Push(x float64) {
	if r.count < r.size {
		r.buf[r.idx] = x
		r.idx = (r.idx + 1) % r.size
		r.count++
		r.sum += x
		r.sumSq += x * x
		return
	}
	old := r.buf[r.idx]
	r.buf[r.idx] = x
	r.idx = (r.idx + 1) % r.size
	r.sum += x - old
	r.sumSq += x*x - old*old
}

func (r *Ring) Mean() float64 {
	if r.count == 0 {
		return 0
	}
	return r.sum / float64(r.count)
}

func (r *Ring) Variance() float64 {
	if r.count < 2 {
		return 0
	}
	n := float64(r.count)
	mean := r.Mean()
	v := (r.sumSq / n) - (mean * mean)
	if v < 0 {
		return 0
	}
	return v
}

func (r *Ring) Count() int {
	return r.count
}

func (r *Ring) SumSq() float64 {
	return r.sumSq
}

func (r *Ring) Std() float64 {
	v := r.Variance()
	if v <= 0 {
		return 0
	}
	return math.Sqrt(v)
}

func (r *Ring) AtLag(lag int) (float64, bool) {
	if r.count == 0 || lag < 0 || lag >= r.count {
		return 0, false
	}
	last := (r.idx - 1 + r.size) % r.size
	pos := (last - lag + r.size) % r.size
	return r.buf[pos], true
}
