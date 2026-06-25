package greetings

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Hello returns a random greeting for the named person.
// (Kept for the package's tests — embeds the name, errors on empty.)
func Hello(name string) (string, error) {
	if name == "" {
		return "", errors.New("empty name")
	}
	return fmt.Sprintf(randomFormat(), name), nil
}

// Greet returns a greeting tailored to the time of day, mixing English
// and Nigerian Pidgin. hour is the caller's local hour (0-23). It also
// returns the part of day ("morning"/"afternoon"/"evening"/"night").
func Greet(name string, hour int) (message, partOfDay string, err error) {
	if strings.TrimSpace(name) == "" {
		return "", "", errors.New("please enter a name")
	}
	part := timeOfDay(hour)
	prefix := timeGreeting(part)
	body := fmt.Sprintf(randomFormat(), strings.TrimSpace(name))
	return fmt.Sprintf("%s — %s", prefix, body), part, nil
}

// timeOfDay buckets an hour (0-23) into a part of the day.
func timeOfDay(hour int) string {
	switch {
	case hour >= 5 && hour < 12:
		return "morning"
	case hour >= 12 && hour < 17:
		return "afternoon"
	case hour >= 17 && hour < 21:
		return "evening"
	default:
		return "night"
	}
}

// timeGreeting returns a time-appropriate opener (English + Pidgin).
func timeGreeting(part string) string {
	switch part {
	case "morning":
		return pick("Good morning", "Morning o", "Rise and shine")
	case "afternoon":
		return pick("Good afternoon", "Afternoon o", "How the day dey go")
	case "evening":
		return pick("Good evening", "Evening o", "How work")
	default:
		return pick("Good evening", "Still dey up?", "How night")
	}
}

// randomFormat returns one of a set of greeting messages (English + Pidgin).
// Every format embeds the name, so Hello's tests still pass.
func randomFormat() string {
	formats := []string{
		// English
		"Hi, %v. Welcome!",
		"Great to see you, %v!",
		"Hail, %v! Well met!",
		"Welcome aboard, %v!",
		// Nigerian Pidgin
		"How far, %v!",
		"%v, how body?",
		"Wetin dey happen, %v!",
		"%v, you welcome o!",
		"Oya %v, na you we dey wait!",
		"Abeg %v, how you dey?",
	}
	return formats[rand.Intn(len(formats))]
}

func pick(opts ...string) string {
	return opts[rand.Intn(len(opts))]
}

// init sets initial values for variables used in the function.
func init() {
	rand.Seed(time.Now().UnixNano())
}
