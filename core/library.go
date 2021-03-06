package core

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"
)

type stcoreError struct {
	s string
}

func (e *stcoreError) Error() string {
	log.Println(e.s)
	return e.s
}

func NewError(s string) *stcoreError {
	return &stcoreError{
		s: s,
	}
}

// Library is the set of all core block Specs
func GetLibrary() map[string]Spec {
	return map[string]Spec{
		// mechanisms
		"delay":    Delay(),
		"set":      Set(),
		"log":      Log(),
		"sink":     Sink(),
		"latch":    Latch(),
		"gate":     Gate(),
		"identity": Identity(),
		"append":   Append(),
		"tail":     Tail(),
		"head":     Head(),
		// dyads
		"+":   Addition(),
		"-":   Subtraction(),
		"×":   Multiplication(),
		"÷":   Division(),
		"^":   Exponentiation(),
		"mod": Modulation(),
		">":   GreaterThan(),
		"<":   LessThan(),
		"==":  EqualTo(),
		"!=":  NotEqualTo(),
		// random sources
		"uniform":   UniformRandom(),
		"normal":    NormalRandom(),
		"Zipf":      ZipfRandom(),
		"Poisson":   PoissonRandom(),
		"Bernoulli": BernoulliRandom(),
		// keyvalue
		"kvGet":    kvGet(),
		"kvSet":    kvSet(),
		"kvClear":  kvClear(),
		"kvDump":   kvDump(),
		"kvDelete": kvDelete(),
		// stateful
		"first": First(),
	}
}

func First() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"in"}},
		Outputs: []Pin{Pin{"first"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			_, ok := internal[0]
			if !ok {
				out[0] = true
				internal[0] = struct{}{}
			} else {
				out[0] = false
			}
			return nil
		},
	}
}

// Delay emits the message on passthrough after the specified duration
func Delay() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"passthrough"}, Pin{"duration"}},
		Outputs: []Pin{Pin{"passthrough"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			t, err := time.ParseDuration(in[1].(string))
			if err != nil {
				out[0] = err
				return nil
			}
			timer := time.NewTimer(t)
			select {
			case <-timer.C:
				out[0] = in[0]
				return nil
			case f := <-i:
				return f
			}
			return nil
		},
	}
}

// Set creates a new message with the specified key and value
func Set() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"key"}, Pin{"value"}},
		Outputs: []Pin{Pin{"object"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			out[0] = map[string]interface{}{
				in[0].(string): in[1],
			}
			return nil
		},
	}
}

// Log writes the inbound message to stdout
// TODO where should this write exactly?
func Log() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"log"}},
		Outputs: []Pin{},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			o, err := json.Marshal(in[0])
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(string(o))
			return nil
		},
	}
}

// Sink discards the inbound message
func Sink() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"in"}},
		Outputs: []Pin{},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			return nil
		},
	}
}

// Latch emits the inbound message on the 0th output if ctrl is true,
// and the 1st output if ctrl is false
func Latch() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"in"}, Pin{"ctrl"}},
		Outputs: []Pin{Pin{"out"}, Pin{"out"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			controlSignal, ok := in[1].(bool)
			if !ok {
				out[0] = NewError("Latch ctrl requires bool")
				return nil
			}
			if controlSignal {
				out[0] = in[0]
				out[1] = nil
			} else {
				out[1] = in[0]
				out[0] = nil
			}
			return nil
		},
	}
}

// Gate emits the inbound message upon receiving a message on its trigger
func Gate() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"in"}, Pin{"ctrl"}},
		Outputs: []Pin{Pin{"out"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			out[0] = in[0]
			return nil
		},
	}
}

// Identity emits the inbound message immediately
func Identity() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"in"}},
		Outputs: []Pin{Pin{"out"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			out[0] = in[0]
			return nil
		},
	}
}

// Head emits the first element of the inbound array on one output,
//and the tail of the array on the other.
func Head() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"in"}},
		Outputs: []Pin{Pin{"head"}, Pin{"tail"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			arr, ok := in[0].([]interface{})
			if !ok {
				out[0] = NewError("head requires an array")
				return nil
			}
			out[0] = arr[0]
			out[1] = arr[1:len(arr)]
			return nil
		},
	}
}

// Tail emits the last element of the inbound array on one output,
//and the head of the array on the other.
func Tail() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"in"}},
		Outputs: []Pin{Pin{"tail"}, Pin{"head"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			arr, ok := in[0].([]interface{})
			if !ok {
				out[0] = NewError("tail requires an array")
				return nil
			}
			out[0] = arr[len(arr)]
			out[1] = arr[0 : len(arr)-1]
			return nil
		},
	}
}

// Append appends the supplied element to the supplied array
func Append() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"element"}, Pin{"array"}},
		Outputs: []Pin{Pin{"array"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			arr, ok := in[1].([]interface{})
			if !ok {
				out[0] = NewError("Append requires an array")
				return nil
			}
			out[0] = append(arr, in[0])
			return nil
		},
	}
}

// Addition returns the sum of the addenda
func Addition() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"addend"}, Pin{"addend"}},
		Outputs: []Pin{Pin{"sum"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			a1, ok := in[0].(float64)
			if !ok {
				out[0] = NewError("Addition requires floats")
				return nil
			}
			a2, ok := in[1].(float64)
			if !ok {
				out[0] = NewError("Addition requires floats")
				return nil
			}
			out[0] = a1 + a2
			return nil
		},
	}
}

// Subtraction returns the difference of the minuend - subtrahend
func Subtraction() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"minuend"}, Pin{"subtrahend"}},
		Outputs: []Pin{Pin{"difference"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			minuend, ok := in[0].(float64)
			if !ok {
				out[0] = NewError("Subtraction requires floats")
				return nil
			}
			subtrahend, ok := in[1].(float64)
			if !ok {
				out[0] = NewError("Subtraction requires floats")
				return nil
			}
			out[0] = minuend - subtrahend
			return nil
		},
	}
}

// Multiplication returns the product of the multiplicanda
func Multiplication() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"multiplicand"}, Pin{"multiplicand"}},
		Outputs: []Pin{Pin{"product"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			m1, ok := in[0].(float64)
			if !ok {
				out[0] = NewError("Multiplication requires floats")
				return nil
			}
			m2, ok := in[1].(float64)
			if !ok {
				out[0] = NewError("Multiplication requires floats")
				return nil
			}
			out[0] = m1 * m2
			return nil
		},
	}
}

// Division returns the quotient of the dividend / divisor
func Division() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"dividend"}, Pin{"divisor"}},
		Outputs: []Pin{Pin{"quotient"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			d1, ok := in[0].(float64)
			if !ok {
				out[0] = NewError("Division requires floats")
				return nil
			}
			d2, ok := in[1].(float64)
			if !ok {
				out[0] = NewError("Division requires floats")
				return nil
			}
			out[0] = d1 / d2
			return nil
		},
	}
}

// Exponentiation returns the base raised to the exponent
func Exponentiation() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"base"}, Pin{"exponent"}},
		Outputs: []Pin{Pin{"power"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			d1, ok := in[0].(float64)
			if !ok {
				out[0] = NewError("Exponentiation requires floats")
				return nil
			}
			d2, ok := in[1].(float64)
			if !ok {
				out[0] = NewError("Exponentiation requires floats")
				return nil
			}
			out[0] = math.Pow(d1, d2)
			return nil
		},
	}
}

// Modulation returns the remainder of the dividend mod divisor
func Modulation() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"dividend"}, Pin{"divisor"}},
		Outputs: []Pin{Pin{"remainder"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			d1, ok := in[0].(float64)
			if !ok {
				out[0] = NewError("Modulation requires floats")
				return nil
			}
			d2, ok := in[1].(float64)
			if !ok {
				out[0] = NewError("Modultion requires floats")
				return nil
			}
			out[0] = math.Mod(d1, d2)
			return nil
		},
	}
}

// GreaterThan returns true if value[0] > value[1] or false otherwise
func GreaterThan() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"value"}, Pin{"value"}},
		Outputs: []Pin{Pin{"IsGreaterThan"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			d1, ok := in[0].(float64)
			if !ok {
				out[0] = NewError("GreaterThan requires floats")
				return nil
			}
			d2, ok := in[1].(float64)
			if !ok {
				out[0] = NewError("GreaterThan requires floats")
				return nil
			}
			out[0] = d1 > d2
			return nil
		},
	}
}

// LessThan returns true if value[0] < value[1] or false otherwise
func LessThan() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"value"}, Pin{"value"}},
		Outputs: []Pin{Pin{"IsLessThan"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			d1, ok := in[0].(float64)
			if !ok {
				out[0] = NewError("LessThan requires floats")
				return nil
			}
			d2, ok := in[1].(float64)
			if !ok {
				out[0] = NewError("LessThan requires floats")
				return nil
			}
			out[0] = d1 < d2
			return nil
		},
	}
}

// EqualTo returns true if value[0] == value[1] or false otherwise
func EqualTo() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"value"}, Pin{"value"}},
		Outputs: []Pin{Pin{"IsEqualTo"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			out[0] = in[0] == in[1]
			return nil
		},
	}
}

// NotEqualTo returns true if value[0] != value[1] or false otherwise
func NotEqualTo() Spec {
	return Spec{
		Inputs:  []Pin{Pin{"value"}, Pin{"value"}},
		Outputs: []Pin{Pin{"IsNotEqualTo"}},
		Kernel: func(in, out, internal MessageMap, s Store, i chan Interrupt) Interrupt {
			out[0] = in[0] != in[1]
			return nil
		},
	}
}
