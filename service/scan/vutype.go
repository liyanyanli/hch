package scan

//	"errors"
import (
	"flag"
)

// Priority defines a vulnerability priority
type Priority string

const (
	// Unknown is either a security problem that has not been
	// assigned to a priority yet or a priority that our system
	// did not recognize
	Unknown Priority = "Unknown"
	// Negligible is technically a security problem, but is
	// only theoretical in nature, requires a very special
	// situation, has almost no install base, or does no real
	// damage. These tend not to get backport from upstreams,
	// and will likely not be included in security updates unless
	// there is an easy fix and some other issue causes an update.
	Negligible Priority = "Negligible"
	// Low is a security problem, but is hard to
	// exploit due to environment, requires a user-assisted
	// attack, a small install base, or does very little damage.
	// These tend to be included in security updates only when
	// higher priority issues require an update, or if many
	// low priority issues have built up.
	Low Priority = "Low"
	// Medium is a real security problem, and is exploitable
	// for many people. Includes network daemon denial of service
	// attacks, cross-site scripting, and gaining user privileges.
	// Updates should be made soon for this priority of issue.
	Medium Priority = "Medium"
	// High is a real problem, exploitable for many people in a default
	// installation. Includes serious remote denial of services,
	// local root privilege escalations, or data loss.
	High Priority = "High"
	// Critical is a world-burning problem, exploitable for nearly all people
	// in a default installation of Linux. Includes remote root
	// privilege escalations, or massive data loss.
	Critical Priority = "Critical"
	// Defcon1 is a Critical problem which has been manually highlighted by
	// the team. It requires an immediate attention.
	Defcon1 Priority = "Defcon1"
)

// Priorities lists all known priorities, ordered from lower to higher
var Priorities = []Priority{Unknown, Negligible, Low, Medium, High, Critical, Defcon1}

var (
	flagMinimumSeverity = flag.String("minimum-severity", "Negligible", "Minimum severity of vulnerabilities to show (Unknown, Negligible, Low, Medium, High, Critical, Defcon1)")
)

func (p Priority) Compare(p2 Priority) int {
	var i1, i2 int

	for i1 = 0; i1 < len(Priorities); i1 = i1 + 1 {
		if p == Priorities[i1] {
			break
		}
	}
	for i2 = 0; i2 < len(Priorities); i2 = i2 + 1 {
		if p2 == Priorities[i2] {
			break
		}
	}

	return i1 - i2
}
