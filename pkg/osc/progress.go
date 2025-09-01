package osc

import (
	"fmt"
	"os"
)

// \x1b]9;4;%d;%d\x1b\\

// 9;4;
// This is the specific command number, 9, followed by a semicolon and the parameter 4.

// 9: Informs the terminal it's a progress bar control.
// 4: Specifies the type of progress control, such as taskbar progress.

// <state>:
// This is the first parameter after the 9;4;.
// 0: Turns off progress.
// 1: Normal progress.
// 2: Error state.
// 3: Indeterminate state (like a spinning loader).
// 4: Paused state.

type State int

const (
	TurnOffProgress    State = 0
	NormalProgress     State = 1
	ErrorState         State = 2
	IndeterminateState State = 3
	PausedState        State = 4

	oscSeq = "\x1b]9;4;%d;%f\x1b\\"
)

func SetOSCProgress(progress float64) {
	state := NormalProgress
	if progress == 100 {
		state = TurnOffProgress
	}
	SetOSC94(state, progress)
}

func SetOSC94(state State, progress float64) {
	_, _ = fmt.Fprintf(os.Stdout, oscSeq, state, progress)
}
