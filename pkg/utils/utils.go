// Copyright 2017 The elasticsearch-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package utils

import "strings"

// String converts an int32 to a string
// Taken from https://stackoverflow.com/a/39444005
func String(n int32) string {
	buf := [11]byte{}
	pos := len(buf)
	i := int64(n)
	signed := i < 0
	if signed {
		i = -i
	}
	for {
		pos--
		buf[pos], i = '0'+byte(i%10), i/10
		if i == 0 {
			if signed {
				pos--
				buf[pos] = '-'
			}
			return string(buf[pos:])
		}
	}
}

// FormatMemoryJava takes a Kubernetes memory string (example 512Mi, 512M or 1G,
// etc.) and converts it to the specific Java memory "type".
func FormatMemoryJava(memory string) string {
	if string(memory[len(memory)-1]) == "i" {
		return memory[:len(memory)-1]
	}
	return strings.ToLower(memory)
}
