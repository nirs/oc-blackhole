// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

type Progress struct {
	mutex       sync.Mutex
	description string
	tasks       uint
	done        uint
	width       int
	out         io.Writer
}

// NewProgress return a new progress indicator.
func NewProgress(descripiton string, tasks uint, out io.Writer) *Progress {
	p := &Progress{
		description: descripiton,
		tasks:       tasks,
		width:       80,
		out:         out,
	}
	p.update()
	return p
}

// SetTasks changes the number of tasks.
func (p *Progress) SetTasks(tasks uint) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.tasks != tasks {
		p.tasks = tasks
		p.update()
	}
}

// SetDescription changes the description.
func (p *Progress) SetDescription(desciption string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.description != desciption {
		p.description = desciption
		p.update()
	}
}

// Add completed tasks to progress.
func (p *Progress) Add(completed uint) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.done < p.tasks {
		p.done = min(p.done+completed, p.tasks)
		p.update()
	}
}

// Clear the progress output.
func (p *Progress) Clear() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	line := strings.Repeat(" ", p.width-1) + "\r"
	p.out.Write([]byte(line))
}

func (p *Progress) update() {
	// Padd output to full line to cover previous line data
	padding := strings.Repeat(" ", p.width-1-len(p.description)-len("[ ---- ] "))

	var line string
	if p.tasks == 0 {
		line = fmt.Sprintf("[ ---- ] %s%s\r", p.description, padding)
	} else {
		value := float64(p.done*100) / float64(p.tasks)
		line = fmt.Sprintf("[ %3.0f%% ] %s%s\r", value, p.description, padding)
	}

	p.out.Write([]byte(line))
}
