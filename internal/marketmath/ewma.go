package marketmath

import "math"

type EWMA struct {
	Lambda float64
	Var    float64
	Init   bool
}

func NewEWMA(lambda float64) *EWMA {
	return &EWMA{Lambda: lambda}
}

func (e *EWMA) Update(ret float64) {
	x2 := ret * ret
	if !e.Init {
		e.Var = x2
		e.Init = true
		return
	}
	e.Var = e.Lambda*e.Var + (1.0-e.Lambda)*x2
}

func (e *EWMA) Vol() float64 {
	return math.Sqrt(math.Max(e.Var, 1e-12))
}
