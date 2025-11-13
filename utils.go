package main

import "strings"

func removeProfanity(s string) string {
	bad := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}
	parts := strings.Split(s, " ")
	for i, p := range parts {
		if _, ok := bad[strings.ToLower(p)]; ok {
			parts[i] = "****"
		}
	}
	return strings.Join(parts, " ")
}
