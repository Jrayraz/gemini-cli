package command

import (
	"fmt"
	"strings"
)

// Command represents a parsed command with its name, arguments, and flags.
type Command struct {
	Name string
	Args []string
	Flags map[string]string
}

// Parse takes a raw command string and converts it into a Command struct.
// This is a basic implementation and will be refined to handle more complex scenarios.
func Parse(rawCommand string) (*Command, error) {
	if rawCommand == "" {
		return nil, fmt.Errorf("command string cannot be empty")
	}

	parts := strings.Fields(rawCommand)
	if len(parts) == 0 {
		return nil, fmt.Errorf("command string is empty after trimming")
	}

	cmd := &Command{
		Name:  parts[0],
		Args:  []string{},
		Flags: make(map[string]string),
	}

	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if strings.HasPrefix(part, "--") {
			// Long flag (e.g., --flag=value or --flag value)
			flagParts := strings.SplitN(part[2:], "=", 2)
			flagName := flagParts[0]
			if len(flagParts) == 2 {
				cmd.Flags[flagName] = flagParts[1]
			} else {
				// Flag without value or flag with next part as value
				if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "-") {
					cmd.Flags[flagName] = parts[i+1]
					i++ // Consume next part as value
				} else {
					cmd.Flags[flagName] = "true" // Assume boolean flag
				}
			}
		} else if strings.HasPrefix(part, "-") {
			// Short flag (e.g., -f value or -f=value)
			flagParts := strings.SplitN(part[1:], "=", 2)
			flagName := flagParts[0]
			if len(flagParts) == 2 {
				cmd.Flags[flagName] = flagParts[1]
			} else {
				// Flag without value or flag with next part as value
				if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "-") {
					cmd.Flags[flagName] = parts[i+1]
					i++ // Consume next part as value
				} else {
					cmd.Flags[flagName] = "true" // Assume boolean flag
				}
			}
		} else {
			// Argument
			cmd.Args = append(cmd.Args, part)
		}
	}

	return cmd, nil
}
